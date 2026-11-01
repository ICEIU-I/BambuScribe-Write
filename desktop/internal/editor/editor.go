// Package editor applies file-oriented editing tools to the same project model
// used by the canvas. Generated geometry is always rebuilt by engine.Font.
package editor

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type Operation struct {
	Tool   string          `json:"tool"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Result struct {
	Tool string `json:"tool"`
	ID   string `json:"id,omitempty"`
}

// Decode rejects misspelled fields and trailing JSON instead of silently
// executing a different operation from the one the caller intended.
func Decode(raw []byte, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("参数不能为 null")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return fmt.Errorf("参数 JSON 无效：%w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("参数只能包含一个 JSON 值")
	}
	return nil
}

// Clone copies editable slices while sharing immutable raster data strings.
func Clone(p project.Project) project.Project {
	if p.Writing.PenUp != nil {
		v := *p.Writing.PenUp
		p.Writing.PenUp = &v
	}
	if p.Writing.PenDown != nil {
		v := *p.Writing.PenDown
		p.Writing.PenDown = &v
	}
	p.Erasures = append([]engine.PageErasure(nil), p.Erasures...)
	for i := range p.Erasures {
		p.Erasures[i].Path = append([]engine.Point(nil), p.Erasures[i].Path...)
		if p.Erasures[i].Cuts != nil {
			cuts := append([]engine.InkCut{}, (*p.Erasures[i].Cuts)...)
			for j := range cuts {
				cuts[j].Ranges = append([][2]float64(nil), cuts[j].Ranges...)
			}
			p.Erasures[i].Cuts = &cuts
		}
		if p.Erasures[i].ObjectIDs != nil {
			ids := append([]string{}, (*p.Erasures[i].ObjectIDs)...)
			p.Erasures[i].ObjectIDs = &ids
		}
	}
	p.Tables = append([]engine.TableAsset(nil), p.Tables...)
	for i, t := range p.Tables {
		t.ColumnWidths = append([]float64(nil), t.ColumnWidths...)
		t.RowHeights = append([]float64(nil), t.RowHeights...)
		t.Cells = append([]engine.TableCell(nil), t.Cells...)
		for j, c := range t.Cells {
			if c.Borders != nil {
				b := *c.Borders
				for _, field := range []**bool{&b.Top, &b.Right, &b.Bottom, &b.Left} {
					if *field != nil {
						v := **field
						*field = &v
					}
				}
				t.Cells[j].Borders = &b
			}
		}
		p.Tables[i] = t
	}
	p.Images = append([]engine.ImageAsset(nil), p.Images...)
	for i := range p.Images {
		p.Images[i].TextRegions = append([]engine.ImageTextRegion(nil), p.Images[i].TextRegions...)
	}
	return p
}

// Apply is transactional: failures leave the input and on-disk project intact.
// undo/redo operate on preceding edits in this batch; the caller retains input
// files for history across separate invocations.
func Apply(source project.Project, ops []Operation) (project.Project, []Result, error) {
	if len(ops) == 0 || len(ops) > 200 {
		return source, nil, fmt.Errorf("一次批处理须包含 1–200 个操作")
	}
	current := Clone(source)
	for _, e := range current.Erasures {
		if e.Cuts != nil {
			continue
		}
		font, err := engine.EmbeddedFont()
		if err != nil {
			return source, nil, err
		}
		doc, err := font.LayoutWithErasures(current.Text, current.Layout, current.Images, current.Tables, current.Erasures)
		if err != nil {
			return source, nil, err
		}
		current.Erasures = doc.Erasures
		break
	}
	var past, future []project.Project
	results := make([]Result, 0, len(ops))
	for i, op := range ops {
		var id string
		var err error
		switch op.Tool {
		case "undo", "redo":
			err = Decode(op.Params, &struct{}{})
			if err == nil && op.Tool == "undo" {
				if len(past) == 0 {
					err = fmt.Errorf("没有可撤销的操作")
				} else {
					future = append(future, current)
					current = past[len(past)-1]
					past = past[:len(past)-1]
				}
			} else if err == nil {
				if len(future) == 0 {
					err = fmt.Errorf("没有可重做的操作")
				} else {
					past = append(past, current)
					current = future[len(future)-1]
					future = future[:len(future)-1]
				}
			}
		default:
			next := Clone(current)
			id, err = edit(&next, op)
			if err == nil {
				discardReplacedInk(current, &next)
			}
			if err == nil {
				err = next.Layout.Validate()
			}
			if err == nil {
				err = engine.ValidateTables(next.Tables)
			}
			if err == nil {
				past = append(past, current)
				current = next
				future = nil
			}
		}
		if err != nil {
			return source, nil, fmt.Errorf("操作 %d (%s)：%w", i+1, op.Tool, err)
		}
		results = append(results, Result{Tool: op.Tool, ID: id})
	}
	return current, results, nil
}

func edit(p *project.Project, op Operation) (string, error) {
	switch op.Tool {
	case "text.set":
		var a struct {
			Text *string `json:"text"`
		}
		if err := Decode(op.Params, &a); err != nil {
			return "", err
		}
		if a.Text == nil {
			return "", fmt.Errorf("text.set 必须指定 text")
		}
		p.Erasures = engine.RebaseBodyCuts(p.Erasures, p.Text, *a.Text)
		p.Text = *a.Text
	case "layout.set":
		if err := Decode(op.Params, &p.Layout); err != nil {
			return "", err
		}
	case "writing.set":
		if err := Decode(op.Params, &p.Writing); err != nil {
			return "", err
		}
		return "", p.Writing.Validate()
	case "table.insert", "textbox.insert":
		return insertTable(p, op)
	case "table.update", "textbox.update", "table.cells", "table.merge", "table.split",
		"table.row.insert", "table.row.delete", "table.column.insert", "table.column.delete", "table.paste":
		return editTable(p, op)
	case "image.insert", "image.update", "image.text.insert", "image.text.update", "image.text.delete",
		"image.erase.rect", "image.erase.path", "image.erase.delete":
		return editImage(p, op)
	case "object.move", "object.resize", "object.copy", "object.delete":
		return editObject(p, op)
	default:
		return "", fmt.Errorf("未知工具 %q，使用 tools 查看工具列表", op.Tool)
	}
	return "", nil
}
