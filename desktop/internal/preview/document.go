// Package preview shares repeated vectors in the editor response. The cached
// engine document and exported coordinates remain unchanged.
package preview

import "bambuscribe/desktop/internal/engine"

type Document struct {
	Document document    `json:"document"`
	Strokes  [][]float64 `json:"strokes"`
}

type document struct {
	*engine.Document
	Pages []page `json:"pages"`
}
type page struct {
	*engine.Page
	Strokes     []int   `json:"strokes"`
	TextStrokes []int   `json:"textStrokes"`
	Images      []image `json:"images"`
	Tables      []table `json:"tables,omitempty"`
	Ink         []ink   `json:"ink,omitempty"`
}
type image struct {
	*engine.ImagePlacement
	Strokes        []int    `json:"strokes,omitempty"`
	GraphicStrokes []int    `json:"graphicStrokes"`
	SourceStrokes  []int    `json:"sourceStrokes"`
	TextRegions    []region `json:"textRegions"`
}
type region struct {
	*engine.ImageTextRegion
	Strokes       []int `json:"strokes,omitempty"`
	SourceStrokes []int `json:"sourceStrokes"`
}
type table struct {
	*engine.TablePlacement
	Strokes []int  `json:"strokes"`
	Cells   []cell `json:"cells"`
}
type cell struct {
	*engine.TableCellPlacement
	Strokes []int `json:"strokes"`
}
type ink struct {
	*engine.InkLayer
	Strokes []int `json:"strokes"`
}

// Most repeated strokes share their backing slice between final page ink,
// body ink and editable layers. Pointer identity avoids hashing every point.
type strokeKey struct {
	first *engine.Point
	count int
}

func Pack(doc *engine.Document) *Document {
	out := &Document{Document: document{Document: doc}, Strokes: [][]float64{}}
	seen := map[strokeKey]int{}
	refs := func(strokes []engine.Stroke) []int {
		if strokes == nil {
			return nil
		}
		indexes := make([]int, len(strokes))
		for i, stroke := range strokes {
			key := strokeKey{count: len(stroke)}
			if len(stroke) > 0 {
				key.first = &stroke[0]
			}
			index, ok := seen[key]
			if !ok {
				index = len(out.Strokes)
				seen[key] = index
				coordinates := make([]float64, 0, len(stroke)*2)
				for _, p := range stroke {
					coordinates = append(coordinates, p.X, p.Y)
				}
				out.Strokes = append(out.Strokes, coordinates)
			}
			indexes[i] = index
		}
		return indexes
	}
	for pi := range doc.Pages {
		p := &doc.Pages[pi]
		packed := page{Page: p, Strokes: refs(p.Strokes), TextStrokes: refs(p.TextStrokes)}
		if p.Images != nil {
			packed.Images = make([]image, 0, len(p.Images))
		}
		for ii := range p.Images {
			im := &p.Images[ii]
			packedImage := image{ImagePlacement: im, Strokes: refs(im.Strokes), GraphicStrokes: refs(im.GraphicStrokes), SourceStrokes: refs(im.SourceStrokes)}
			if im.TextRegions != nil {
				packedImage.TextRegions = make([]region, 0, len(im.TextRegions))
			}
			for ri := range im.TextRegions {
				r := &im.TextRegions[ri]
				packedImage.TextRegions = append(packedImage.TextRegions, region{ImageTextRegion: r, Strokes: refs(r.Strokes), SourceStrokes: refs(r.SourceStrokes)})
			}
			packed.Images = append(packed.Images, packedImage)
		}
		for ti := range p.Tables {
			t := &p.Tables[ti]
			packedTable := table{TablePlacement: t, Strokes: refs(t.Strokes)}
			if t.Cells != nil {
				packedTable.Cells = make([]cell, 0, len(t.Cells))
			}
			for ci := range t.Cells {
				c := &t.Cells[ci]
				packedTable.Cells = append(packedTable.Cells, cell{TableCellPlacement: c, Strokes: refs(c.Strokes)})
			}
			packed.Tables = append(packed.Tables, packedTable)
		}
		for li := range p.Ink {
			l := &p.Ink[li]
			packed.Ink = append(packed.Ink, ink{InkLayer: l, Strokes: refs(l.Strokes)})
		}
		out.Document.Pages = append(out.Document.Pages, packed)
	}
	return out
}
