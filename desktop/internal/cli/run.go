// Package cli exposes project editing, rendering and device transfer without
// launching Wails or reading the desktop draft/connection configuration.
package cli

import (
	"bambuscribe/desktop/internal/editor"
	"bambuscribe/desktop/internal/printer"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

type Runner struct {
	NewLink  func() printer.Link
	Discover func(context.Context, string) (string, error)
	Getenv   func(string) string
}

func New() *Runner {
	return &Runner{NewLink: func() printer.Link { return &printer.Network{} }, Discover: printer.Discover, Getenv: os.Getenv}
}

type options struct {
	project, output, params, paramsFile, opsFile, format, page, config, ip, serial string
	image, instruction, session                                                    string
	full                                                                           bool
	timeout                                                                        time.Duration
}

const usage = `BambuScribe CLI
  bambuscribe new --output note.ccf [--params '{"text":"实验"}']
  bambuscribe inspect --project note.ccf [--full]
  bambuscribe tool <tool-name> --project note.ccf --output edited.ccf --params '{...}'
  bambuscribe batch --project note.ccf --output edited.ccf --ops-file operations.json
  bambuscribe preview --project note.ccf --format svg --page 1 --output page.svg
  bambuscribe check --project note.ccf
  bambuscribe export --project note.ccf --page 1|all --output page.gcode|pages.zip
  bambuscribe device discover --ip 192.168.1.100
  bambuscribe device connect|status|watch --config device.json [--timeout 30s]
  bambuscribe device upload --config device.json --project note.ccf --params '{"prefix":"31","pages":[1]}' [--timeout 10m]
  bambuscribe recognize-table --image table.png --output extraction.json
  bambuscribe import-table --project note.ccf --output edited.ccf --params-file import.json
  bambuscribe live snapshot
  bambuscribe live import --params-file import.json
  bambuscribe tools

参数支持 --params JSON 或 --params-file 文件；文件名 - 从 stdin 读取。
批处理为 [{"tool":"textbox.insert","params":{...}}, ...]，支持 undo/redo。
纸面位置和尺寸单位 mm，页码从 1 开始；单元格行列从 0 开始。
设备访问码可放在 BAMBUSCRIBE_ACCESS_CODE；Ctrl+C 取消本进程上传并断开。
结果为 stdout JSON；进度为 stderr JSONL，错误返回非零退出码。
`

func (r *Runner) Run(ctx context.Context, args []string, in io.Reader, out, progress io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(out, usage)
		return err
	}
	command := args[0]
	args = args[1:]
	name := ""
	if command == "tool" || command == "device" || command == "live" {
		if len(args) == 0 {
			return fmt.Errorf("请指定工具或设备操作，使用 help 查看调用方式")
		}
		name, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var o options
	fs.StringVar(&o.image, "image", "", "表格图片文件")
	fs.StringVar(&o.instruction, "instruction", "", "识别补充要求")
	fs.StringVar(&o.session, "session", "", "客户端 editor-session.json")
	fs.StringVar(&o.project, "project", "", "输入 .ccf 项目")
	fs.StringVar(&o.output, "output", "", "输出路径；相同路径表示明确覆盖")
	fs.StringVar(&o.params, "params", "", "JSON 参数")
	fs.StringVar(&o.paramsFile, "params-file", "", "JSON 参数文件；- 为 stdin")
	fs.StringVar(&o.opsFile, "ops-file", "", "批处理文件；- 为 stdin")
	fs.StringVar(&o.format, "format", "json", "preview 格式：json 或 svg")
	fs.StringVar(&o.page, "page", "1", "页码，export 支持 all")
	fs.BoolVar(&o.full, "full", false, "inspect 包含图片原始 data URL")
	fs.StringVar(&o.config, "config", "", "设备配置 JSON 文件")
	fs.StringVar(&o.ip, "ip", "", "发现设备的 IP")
	fs.StringVar(&o.serial, "serial", "", "覆盖配置中的序列号")
	defaultTimeout := 30 * time.Second
	if command == "recognize-table" {
		defaultTimeout = 3 * time.Minute
	}
	if command == "device" && name == "upload" {
		defaultTimeout = 10 * time.Minute
	}
	fs.DurationVar(&o.timeout, "timeout", defaultTimeout, "设备调用最长时间，如 30s 或 10m")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			_, err = io.WriteString(out, usage)
			return err
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("多余参数：%v", fs.Args())
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if command == "tools" {
		return emit(out, Catalog())
	}
	if command == "device" {
		return r.device(ctx, name, o, in, out, progress)
	}
	if command == "recognize-table" || command == "import-table" || command == "live" {
		return r.tableImport(ctx, command, name, o, in, out)
	}
	return r.document(ctx, command, name, o, in, out)
}

func emit(out io.Writer, v any) error { return json.NewEncoder(out).Encode(v) }
func readInput(path string, in io.Reader) ([]byte, error) {
	if path != "-" {
		return editor.ReadFile(path, 32<<20)
	}
	b, err := io.ReadAll(io.LimitReader(in, (32<<20)+1))
	if len(b) > 32<<20 {
		return nil, fmt.Errorf("输入超过 32 MB")
	}
	return b, err
}
func params(o options, in io.Reader) ([]byte, error) {
	if o.params != "" && o.paramsFile != "" {
		return nil, fmt.Errorf("--params 和 --params-file 只能选一个")
	}
	if o.paramsFile != "" {
		return readInput(o.paramsFile, in)
	}
	if len(o.params) > 32<<20 {
		return nil, fmt.Errorf("参数超过 32 MB")
	}
	if o.params == "" {
		return []byte("{}"), nil
	}
	return []byte(o.params), nil
}
