package cli

import (
	"bambuscribe/desktop/internal/editor"
	"bambuscribe/desktop/internal/output"
	"bambuscribe/desktop/internal/printer"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

func (r *Runner) deviceConfig(ctx context.Context, o options, in io.Reader) (printer.Config, error) {
	var c printer.Config
	if o.config == "" {
		return c, fmt.Errorf("请用 --config 指定设备配置 JSON（ip、serial、可选 accessCode）")
	}
	b, err := readInput(o.config, in)
	if err != nil {
		return c, err
	}
	if err = editor.Decode(b, &c); err != nil {
		return c, err
	}
	c.IP = strings.TrimSpace(c.IP)
	c.Serial = strings.ToUpper(strings.TrimSpace(c.Serial))
	if o.serial != "" {
		c.Serial = strings.ToUpper(strings.TrimSpace(o.serial))
	}
	if secret := r.Getenv("BAMBUSCRIBE_ACCESS_CODE"); secret != "" {
		c.AccessCode = secret
	}
	if c.Serial == "" {
		c.Serial, err = r.Discover(ctx, c.IP)
		if err != nil {
			return c, err
		}
	}
	return c, c.Validate()
}
func redact(s, secret string) string {
	if secret != "" {
		return strings.ReplaceAll(s, secret, "[redacted]")
	}
	return s
}

func (r *Runner) device(ctx context.Context, name string, o options, in io.Reader, out, progress io.Writer) (err error) {
	if o.timeout <= 0 {
		return fmt.Errorf("--timeout 必须大于 0")
	}
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	if name == "discover" {
		serial, err := r.Discover(ctx, o.ip)
		if err != nil {
			return err
		}
		return emit(out, map[string]any{"ok": true, "ip": o.ip, "serial": serial})
	}
	if name != "connect" && name != "status" && name != "watch" && name != "upload" {
		return fmt.Errorf("设备操作须为 discover、connect、status、watch 或 upload")
	}
	var pages []printer.PageUpload
	var docID, prefix string
	if name == "upload" {
		var a struct {
			Prefix string `json:"prefix"`
			Pages  []int  `json:"pages"`
		}
		b, e := params(o, in)
		if e != nil {
			return e
		}
		if err = editor.Decode(b, &a); err != nil {
			return err
		}
		if err = printer.ValidateUploadPrefix(a.Prefix); err != nil {
			return err
		}
		prefix = a.Prefix
		if a.Pages != nil && len(a.Pages) == 0 {
			return fmt.Errorf("pages 不能为空；省略该字段表示全部页面")
		}
		indices := make([]int, len(a.Pages))
		for i, p := range a.Pages {
			indices[i] = p - 1
		}
		p, e := load(o.project)
		if e != nil {
			return e
		}
		d, e := render(p)
		if e != nil {
			return e
		}
		pages, err = output.PreparePages(d, p.Writing, indices...)
		if err != nil {
			return err
		}
		docID = d.ID
	}
	if o.config == "-" && o.paramsFile == "-" {
		return fmt.Errorf("配置和参数不能同时从 stdin 读取")
	}
	c, err := r.deviceConfig(ctx, o, in)
	// Never echo device credentials, including an error returned by the link.
	defer func() {
		if err != nil && c.AccessCode != "" {
			err = fmt.Errorf("%s: %w", redact(err.Error(), c.AccessCode), redactedCause(err))
		}
	}()
	if err != nil {
		return err
	}
	changed := make(chan struct{}, 1)
	m := printer.NewManager(r.NewLink(), func(printer.State) {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer m.Close()
	connectCtx, connectCancel := context.WithTimeout(ctx, 15*time.Second)
	err = m.Connect(connectCtx, c)
	connectCancel()
	if err != nil {
		return err
	}
	state := func() printer.State { s := m.State(); s.Message = redact(s.Message, c.AccessCode); return s }
	// A successful MQTT handshake alone is insufficient: upload readiness is
	// determined from the first actual device report, as in the desktop app.
	for m.State().DeviceState == "" {
		if !m.State().Connected {
			return fmt.Errorf("打印机连接已断开")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
	if name == "connect" || name == "status" {
		return emit(out, map[string]any{"ok": true, "state": state()})
	}
	if name == "upload" {
		if _, err = m.UploadPages(docID, prefix, pages); err != nil {
			return err
		}
	}
	var last string
	for {
		s := state()
		encoded, _ := json.Marshal(s)
		key := string(encoded)
		if key != last {
			target := progress
			if name == "watch" {
				target = out
			}
			if err = emit(target, map[string]any{"event": "device.state", "state": s}); err != nil {
				return err
			}
			last = key
		}
		if !s.Connected {
			return fmt.Errorf("打印机连接已断开")
		}
		if name == "upload" {
			switch s.Phase {
			case "uploaded":
				return emit(out, map[string]any{"ok": true, "state": s})
			case "failed", "external", "idle":
				return fmt.Errorf("上传中止：%s", s.Message)
			}
		}
		select {
		case <-ctx.Done():
			if name == "upload" {
				_ = m.CancelUpload()
			}
			return ctx.Err()
		case <-changed:
		}
	}
}

// Preserve cancellation/deadline identity without wrapping credential-bearing
// network errors, whose Error text must never reach the final stderr response.
func redactedCause(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return fmt.Errorf("设备调用失败")
}
