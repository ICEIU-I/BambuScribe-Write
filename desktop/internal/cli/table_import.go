package cli

import (
	"bambuscribe/desktop/internal/editor"
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/liveeditor"
	"bambuscribe/desktop/internal/tableimport"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func configDir() string {
	if d := os.Getenv("BAMBUSCRIBE_CONFIG_DIR"); d != "" {
		return d
	}
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "BambuScribeWrite")
}
func (r *Runner) tableImport(ctx context.Context, command, name string, o options, in io.Reader, out io.Writer) error {
	if command == "live" {
		if name != "snapshot" && name != "import" {
			return fmt.Errorf("live 支持 snapshot 或 import")
		}
		path := o.session
		if path == "" {
			path = filepath.Join(configDir(), "editor-session.json")
		}
		b, err := params(o, in)
		if err != nil {
			return err
		}
		if name == "snapshot" {
			b, _ = json.Marshal(map[string]bool{"full": o.full})
		}
		result, err := liveeditor.Call(ctx, path, name, b)
		if err != nil {
			return err
		}
		return emit(out, json.RawMessage(result))
	}
	if command == "recognize-table" {
		raw, err := editor.ReadFile(o.image, 12<<20)
		if err != nil {
			return fmt.Errorf("图片读取失败，请指定 --image PNG/JPEG 文件")
		}
		mime := http.DetectContentType(raw)
		data := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw)
		c, err := tableimport.LoadConfig(configDir())
		if err != nil {
			return err
		}
		if o.config != "" {
			b, err := editor.ReadFile(o.config, 64<<10)
			if err != nil {
				return err
			}
			if err = editor.Decode(b, &c); err != nil {
				return err
			}
		}
		if key := r.Getenv("BAMBUSCRIBE_AGENT_KEY"); key != "" {
			c.APIKey = key
		}
		callCtx, cancel := context.WithTimeout(ctx, o.timeout)
		defer cancel()
		result, err := tableimport.Recognize(callCtx, c, data, o.instruction)
		if err != nil {
			return err
		}
		if o.output == "" || o.output == "-" {
			return emit(out, result)
		}
		b, _ := json.MarshalIndent(result, "", "  ")
		if err = write(o.output, b, true); err != nil {
			return err
		}
		return emit(out, map[string]any{"ok": true, "path": o.output, "tables": len(result.Tables)})
	}
	p, err := load(o.project)
	if err != nil {
		return err
	}
	b, err := params(o, in)
	if err != nil {
		return err
	}
	var req struct {
		Extraction tableimport.Extraction `json:"extraction"`
		Placement  tableimport.Placement  `json:"placement"`
	}
	if err = editor.Decode(b, &req); err != nil {
		return err
	}
	f, err := engine.EmbeddedFont()
	if err != nil {
		return err
	}
	plan, err := tableimport.Build(f, p, req.Extraction, req.Placement)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	p.Tables = append(p.Tables, plan.Objects...)
	if err = save(p, o.output); err != nil {
		return err
	}
	return emit(out, map[string]any{"ok": true, "path": o.output, "plan": plan})
}
