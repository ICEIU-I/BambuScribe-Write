package engine

import "fmt"

// Source keys exclude placement, dimensions and font size. Thus cuts follow
// existing vectors through moves, scaling and wrapping, but never new content.
func appendObjectInk(doc *Document, images []ImageAsset, tables []TableAsset) {
	imageSources := map[string]string{}
	for _, a := range images {
		imageSources[a.ID] = inkSource(struct {
			Data    string
			Diagram any
			Mode    string
		}{a.Data, a.Diagram, a.Mode})
	}
	tableSources := map[string]string{}
	for _, t := range tables {
		shape := t
		shape.ID = ""
		shape.GroupID = ""
		shape.X = 0
		shape.Y = 0
		shape.Page = 0
		shape.FontSize = 0
		shape.Padding = 0
		shape.Align = ""
		shape.VerticalAlign = ""
		shape.ColumnWidths = make([]float64, len(t.ColumnWidths))
		shape.RowHeights = make([]float64, len(t.RowHeights))
		shape.Cells = append([]TableCell(nil), t.Cells...)
		for i := range shape.Cells {
			shape.Cells[i].Text = ""
			shape.Cells[i].Align = ""
			shape.Cells[i].VerticalAlign = ""
		}
		tableSources[t.ID] = inkSource(shape)
	}
	for pi := range doc.Pages {
		p := &doc.Pages[pi]
		for _, im := range p.Images {
			p.Ink = append(p.Ink, InkLayer{im.ID, "graphic", imageSources[im.ID], im.SourceStrokes})
			for _, r := range im.TextRegions {
				p.Ink = append(p.Ink, InkLayer{im.ID, "region:" + r.ID, inkSource([]string{doc.Settings.FontID, r.Text, r.TextFormat, r.Mode}), r.SourceStrokes})
			}
		}
		for _, t := range p.Tables {
			count := len(t.Strokes)
			for _, c := range t.Cells {
				count -= len(c.Strokes)
			}
			add := func(part, source string, strokes []Stroke) {
				placed := make([]Stroke, len(strokes))
				for i, s := range strokes {
					placed[i] = make(Stroke, len(s))
					for j, v := range s {
						placed[i][j] = Point{v.X + t.X, v.Y + t.Y}
					}
				}
				p.Ink = append(p.Ink, InkLayer{t.ID, part, source, placed})
			}
			add("grid", tableSources[t.ID], t.Strokes[:count])
			for _, c := range t.Cells {
				add(fmt.Sprintf("cell:%d:%d", c.Row, c.Col), inkSource([]string{doc.Settings.FontID, c.Text}), c.Strokes)
			}
		}
	}
}
func erasedPageInk(page Page, erasures []PageErasure) ([]Stroke, []string) {
	order := newPageWritingOrder(page)
	images := map[string]ImagePlacement{}
	for _, im := range page.Images {
		images[im.ID] = im
	}
	for _, layer := range page.Ink {
		strokes := cutLayer(layer, erasures)
		if im, ok := images[layer.ObjectID]; ok {
			asset := ImageAsset{X: im.X, Y: im.Y, Width: im.Width, Height: im.Height, Rotation: im.Rotation}
			regions := im.TextRegions
			if layer.Part != "graphic" {
				for i, r := range regions {
					if layer.Part == "region:"+r.ID {
						regions = regions[i+1:]
						break
					}
				}
			}
			strokes = eraseImageStrokes(strokes, asset, regions)
		}
		order.add(layer, strokes)
	}
	return order.strokes()
}
