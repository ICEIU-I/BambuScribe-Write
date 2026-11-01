package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"fmt"
	"math"
)

type rect struct{ x, y, w, h float64 }

func overlap(a, b rect, gap float64) bool {
	return a.x < b.x+b.w+gap && b.x < a.x+a.w+gap && a.y < b.y+b.h+gap && b.y < a.y+a.h+gap
}

// Build plans additions against the latest complete project. It never replaces
// existing objects. Manual positions reflow body text. Oversized additions use
// the largest font that fits the page width and report any reduction.
func Build(f *engine.Font, p project.Project, e Extraction, anchor Placement) (Plan, error) {
	out := Plan{Objects: []engine.TableAsset{}, Warnings: []string{}, Pages: []int{}, Groups: []Group{}, Preview: []engine.TablePlacement{}}
	if err := e.Validate(); err != nil {
		return out, err
	}
	if err := p.Layout.Validate(); err != nil {
		return out, err
	}
	font, err := f.ForLayout(p.Layout)
	if err != nil {
		return out, err
	}
	f = font
	if anchor.Page == 0 {
		anchor.Page = 1
	}
	if anchor.Page < 1 || anchor.Page > 20001 || math.IsNaN(anchor.Y) || math.IsInf(anchor.Y, 0) || anchor.Y < 0 {
		return out, fmt.Errorf("插入页码或位置无效")
	}
	doc, err := f.LayoutWithObjects(p.Text, p.Layout, p.Images, p.Tables)
	if err != nil {
		return out, err
	}
	s := doc.Settings
	if err := validatePositions(anchor.Positions, len(e.Tables), s); err != nil {
		return out, err
	}
	width := s.PaperWidth - s.MarginLeft - s.MarginRight
	bottom := s.PaperHeight - s.MarginBottom
	occupied := map[int][]rect{}
	found := anchor.AfterID == "" || len(anchor.Positions) > 0
	for pi, page := range doc.Pages {
		for _, c := range page.Cells {
			occupied[pi] = append(occupied[pi], rect{c.X, c.Y, c.Width, c.Height})
		}
		for _, im := range page.Images {
			angle := im.Rotation * math.Pi / 180
			w := math.Abs(im.Width*math.Cos(angle)) + math.Abs(im.Height*math.Sin(angle))
			h := math.Abs(im.Width*math.Sin(angle)) + math.Abs(im.Height*math.Cos(angle))
			occupied[pi] = append(occupied[pi], rect{im.X + (im.Width-w)/2, im.Y + (im.Height-h)/2, w, h})
		}
		for _, t := range page.Tables {
			occupied[pi] = append(occupied[pi], rect{t.X, t.Y, t.Width, t.Height})
			if t.ID == anchor.AfterID {
				anchor.Page = pi + 1
				anchor.Y = t.Y + t.Height + 3
				found = true
			}
		}
	}
	if !found {
		return out, fmt.Errorf("目标表格已删除，请重新选择插入位置")
	}
	page := anchor.Page - 1
	y := max(s.MarginTop, anchor.Y)
	if anchor.AtEnd {
		for _, r := range occupied[page] {
			y = max(y, r.y+r.h+3)
		}
	}
	for i, t := range e.Tables {
		position := Position{Page: page + 1, X: s.MarginLeft, Y: y, Width: width}
		manual := len(anchor.Positions) > 0
		if manual {
			position = anchor.Positions[i]
		}
		position.Width = max(position.Width, min(s.PaperWidth-s.MarginRight-position.X, readableImportWidth(f, t, s)))
		objects, warnings, err := sizeTable(f, t, s, position.Width)
		if err != nil {
			return out, fmt.Errorf("表格 %d：%w", i+1, err)
		}
		position.Width = sum(objects[len(objects)-1].ColumnWidths)
		h := 0.0
		for j, o := range objects {
			h += sum(o.RowHeights)
			if j > 0 {
				h += 2
			}
		}
		if h > bottom-s.MarginTop+1e-8 {
			return out, fmt.Errorf("表格 %d 连同表名高 %.2f mm，超过整页可用高度；请拆分表格后重试", i+1, h)
		}
		position, err = placeImportGroup(i, position, h, s, occupied, manual)
		if err != nil {
			return out, err
		}
		page, y = position.Page-1, position.Y
		occupied[page] = append(occupied[page], rect{position.X, y, position.Width, h})
		group := Group{Position: position, Index: i, Title: normalizeText(t.Title), Height: h, ObjectIDs: []string{}}
		for j := range objects {
			objects[j].GroupID = objects[len(objects)-1].ID
			objects[j].Page = page
			objects[j].X = position.X
			objects[j].Y = y
			group.ObjectIDs = append(group.ObjectIDs, objects[j].ID)
			y += sum(objects[j].RowHeights) + 2
		}
		out.Objects = append(out.Objects, objects...)
		out.Groups = append(out.Groups, group)
		out.Warnings = append(out.Warnings, warnings...)
		out.Pages = append(out.Pages, page+1)
		y++
	}
	combined := append(append([]engine.TableAsset(nil), p.Tables...), out.Objects...)
	if err := engine.ValidateTables(combined); err != nil {
		return out, err
	}
	// Count characters/missing glyphs with the same final native renderer.
	final, err := f.LayoutWithObjects(p.Text, s, p.Images, combined)
	if err != nil {
		return out, err
	}
	ids := map[string]bool{}
	for _, o := range out.Objects {
		ids[o.ID] = true
	}
	expected := map[string]engine.TableAsset{}
	for _, o := range out.Objects {
		expected[o.ID] = o
	}
	for pi, pg := range final.Pages {
		for _, t := range pg.Tables {
			if !ids[t.ID] {
				continue
			}
			want := expected[t.ID]
			if pi != want.Page || math.Abs(t.X-want.X) > 1e-6 || math.Abs(t.Y-want.Y) > 1e-6 {
				return out, fmt.Errorf("文档排版发生变化，请重新调整表格位置")
			}
			out.Preview = append(out.Preview, t)
			if len(t.Issues) > 0 {
				return out, fmt.Errorf("导入排版失败：%s", t.Issues[0])
			}
			for _, c := range t.Cells {
				for _, m := range c.Missing {
					out.Warnings = append(out.Warnings, "字库缺少 "+m.Char+"，请修改后导出")
				}
			}
		}
	}
	return out, nil
}
