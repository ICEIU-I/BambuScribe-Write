package cli

import (
	"bambuscribe/desktop/internal/editor"
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"bambuscribe/desktop/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

func load(path string) (project.Project, error) {
	if path == "" {
		return project.Project{}, fmt.Errorf("请指定 --project .ccf 文件")
	}
	b, err := editor.ReadFile(path, 128<<20)
	if err != nil {
		return project.Project{}, err
	}
	return project.Decode(b)
}
func render(p project.Project) (*engine.Document, error) {
	f, err := engine.EmbeddedFont()
	if err != nil {
		return nil, err
	}
	return f.LayoutWithErasures(p.Text, p.Layout, p.Images, p.Tables, p.Erasures)
}
func write(path string, data []byte, mode bool) error {
	if path == "" || path == "-" {
		return fmt.Errorf("请指定输出文件 --output")
	}
	if mode {
		return storage.WriteAtomic(path, data, 0600)
	}
	return storage.WriteAtomic(path, data, 0644)
}
func save(p project.Project, path string) error {
	if strings.ToLower(filepath.Ext(path)) != ".ccf" {
		return fmt.Errorf("项目输出文件须使用 .ccf 扩展名")
	}
	data, err := project.Encode(p)
	if err != nil {
		return err
	}
	return write(path, data, true)
}
func pageIndex(raw string, d *engine.Document) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > len(d.Pages) {
		return 0, fmt.Errorf("页码须为 1–%d", len(d.Pages))
	}
	return n - 1, nil
}

func (r *Runner) document(ctx context.Context, command, name string, o options, in io.Reader, out io.Writer) error {
	var p project.Project
	var err error
	if command == "new" {
		p = project.Project{Version: project.Version, Layout: engine.DefaultLayout(), Writing: engine.DefaultWriting(), Images: []engine.ImageAsset{}, Tables: []engine.TableAsset{}}
		b, e := params(o, in)
		if e != nil {
			return e
		}
		if err = editor.Decode(b, &p); err != nil {
			return err
		}
	} else {
		p, err = load(o.project)
		if err != nil {
			return err
		}
	}
	switch command {
	case "inspect":
		if !o.full {
			for i := range p.Images {
				p.Images[i].Data = ""
			}
		}
		return emit(out, p)
	case "tool", "batch", "new", "save":
		var results []editor.Result
		if command == "tool" || command == "batch" {
			var ops []editor.Operation
			if command == "tool" {
				b, e := params(o, in)
				if e != nil {
					return e
				}
				ops = []editor.Operation{{Tool: name, Params: b}}
			} else {
				if o.opsFile == "" {
					return fmt.Errorf("批处理需要 --ops-file，- 表示 stdin")
				}
				b, e := readInput(o.opsFile, in)
				if e != nil {
					return e
				}
				if err = editor.Decode(b, &ops); err != nil {
					return err
				}
			}
			p, results, err = editor.Apply(p, ops)
			if err != nil {
				return err
			}
		}
		// Missing glyphs and narrow boxes remain editable and are reported by
		// check/export. Structural failures cannot replace the project file.
		d, err := render(p)
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err = save(p, o.output); err != nil {
			return err
		}
		return emit(out, map[string]any{"ok": true, "path": o.output, "operations": results, "pages": len(d.Pages), "missing": d.Missing})
	case "preview", "check", "export":
		d, err := render(p)
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if command == "check" {
			return check(out, d, p.Writing)
		}
		if command == "export" {
			var data []byte
			ext := ".gcode"
			if o.page == "all" {
				ext = ".zip"
			}
			if strings.ToLower(filepath.Ext(o.output)) != ext {
				return fmt.Errorf("本次导出文件须使用 %s 扩展名", ext)
			}
			if o.page == "all" {
				data, err = engine.ExportAll(d, p.Writing)
			} else {
				i, e := pageIndex(o.page, d)
				if e != nil {
					return e
				}
				data, err = engine.ExportPage(d, i, p.Writing)
			}
			if err != nil {
				return err
			}
			if err = write(o.output, data, false); err != nil {
				return err
			}
			return emit(out, map[string]any{"ok": true, "path": o.output, "bytes": len(data)})
		}
		var data []byte
		if o.output != "" && o.output != "-" && strings.ToLower(filepath.Ext(o.output)) != "."+o.format {
			return fmt.Errorf("预览输出文件须使用 .%s 扩展名", o.format)
		}
		switch o.format {
		case "json":
			if o.output == "" {
				return emit(out, d)
			}
			data, err = json.Marshal(d)
		case "svg":
			i, e := pageIndex(o.page, d)
			if e != nil {
				return e
			}
			data = svgPage(d, i)
		default:
			return fmt.Errorf("预览格式须为 json 或 svg")
		}
		if err != nil {
			return err
		}
		if o.output == "-" {
			_, err = out.Write(data)
			return err
		}
		if err = write(o.output, data, false); err != nil {
			return err
		}
		return emit(out, map[string]any{"ok": true, "path": o.output})
	default:
		return fmt.Errorf("未知命令 %q，使用 help 查看调用方式", command)
	}
}

func check(out io.Writer, d *engine.Document, w engine.WritingSettings) error {
	var pages []map[string]any
	valid := true
	nonempty := 0
	for i, p := range d.Pages {
		var problem string
		if len(p.Strokes) > 0 || len(p.Missing) > 0 || len(p.Issues) > 0 {
			nonempty++
			_, err := engine.ExportPage(d, i, w)
			if err != nil {
				valid = false
				problem = err.Error()
			}
		}
		pages = append(pages, map[string]any{"page": i + 1, "strokes": len(p.Strokes), "issues": p.Issues, "warnings": p.Warnings, "missing": p.Missing, "error": problem})
	}
	if nonempty == 0 {
		valid = false
	}
	if err := emit(out, map[string]any{"ok": valid, "documentId": d.ID, "pages": pages}); err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("项目未通过导出检查，请处理各页 error；空项目没有可书写笔画")
	}
	return nil
}
