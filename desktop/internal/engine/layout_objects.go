package engine

import (
	"fmt"
	"math"
	"sort"
)

const objectLayoutGap = 1.0 // paper millimeters between independent content

// Requested geometry stays in the project. These resolved positions let the
// editor display and move the same objects as the exporter without rewriting
// the draft, or folding automatic movements into unrelated undo steps.
type ObjectPosition struct {
	ID         string       `json:"id"`
	Page       int          `json:"page"`
	X          float64      `json:"x"`
	Y          float64      `json:"y"`
	Source     RegionBounds `json:"source"`
	SourcePage int          `json:"sourcePage"`
	Rotation   float64      `json:"rotation"`
}

type layoutObject struct {
	id, label string
	page      int
	x, y      float64
	w, h      float64
	dx, dy    float64
	rotation  float64
	image     int
	table     int
}

type objectPlan struct {
	images       []ImageAsset
	tables       []TableAsset
	renders      []TablePlacement
	positions    []ObjectPosition
	imageRenders []imageRender
	bodyOccupied map[int][]RegionBounds
}

func (f *Font) planObjects(s LayoutSettings, images []ImageAsset, tables []TableAsset) (objectPlan, error) {
	out := objectPlan{
		images: append([]ImageAsset(nil), images...), tables: append([]TableAsset(nil), tables...),
		renders: make([]TablePlacement, len(tables)), positions: []ObjectPosition{},
	}
	objects := make([]layoutObject, 0, len(images)+len(tables))
	seen := map[string]bool{}
	for i, im := range images {
		if im.Page < 0 || im.Page > maxDocumentCharacters {
			return out, fmt.Errorf("图片页码无效，请将图片放回文档范围内")
		}
		if im.ID == "" || seen[im.ID] {
			return out, fmt.Errorf("图片标识无效或重复，请重新插入图片")
		}
		seen[im.ID] = true
		if !finite(im.X) || !finite(im.Y) || im.X < 0 || im.Y < 0 || !finite(im.Width) || !finite(im.Height) || im.Width < 2 || im.Height < 2 || !finite(im.Rotation) {
			return out, fmt.Errorf("图片尺寸或位置无效")
		}
		a := im.Rotation * math.Pi / 180
		w := math.Abs(im.Width*math.Cos(a)) + math.Abs(im.Height*math.Sin(a))
		h := math.Abs(im.Width*math.Sin(a)) + math.Abs(im.Height*math.Cos(a))
		objects = append(objects, layoutObject{im.ID, "图片", im.Page, im.X, im.Y, w, h, (im.Width - w) / 2, (im.Height - h) / 2, im.Rotation, i, -1})
	}
	for i, t := range tables {
		if seen[t.ID] {
			return out, fmt.Errorf("对象标识重复，请重新插入对象")
		}
		seen[t.ID] = true
		// Measure at the origin; edge checks happen after measuring the object.
		measured := t
		measured.X, measured.Y = 0, 0
		render := f.renderTable(measured, s)
		out.renders[i] = render
		objects = append(objects, layoutObject{t.ID, t.label(), t.Page, t.X, t.Y, render.Width, render.Height, 0, 0, 0, -1, i})
	}
	// Reading order is independent of asset-array order, so old projects, undo
	// and reload always settle on the same positions.
	sort.Slice(objects, func(i, j int) bool {
		a, b := objects[i], objects[j]
		if a.page != b.page {
			return a.page < b.page
		}
		if a.y != b.y {
			return a.y < b.y
		}
		if a.x != b.x {
			return a.x < b.x
		}
		return a.id < b.id
	})
	for _, o := range objects {
		// Explicit object anchors may use the whole sheet, including margins.
		// Only automatic page overflow follows the text margins.
		bottom := s.PaperHeight
		// Include the unrotated frame in edge checks: the image renderer and its
		// controls use that frame, while occupancy uses the rotated bounds.
		frameW, frameH := o.w, o.h
		if o.image >= 0 {
			frameW, frameH = images[o.image].Width, images[o.image].Height
		}
		left, top := min(0, o.dx), min(0, o.dy)
		right, lower := max(frameW, o.dx+o.w), max(frameH, o.dy+o.h)
		flowBottom := max(s.PaperHeight-s.MarginBottom, lower-top)
		if right-left > s.PaperWidth+1e-8 || lower-top > bottom+1e-8 {
			return out, fmt.Errorf("%s尺寸超过整页，请缩小尺寸或拆分后重试", o.label)
		}
		x := max(-left, min(o.x, s.PaperWidth-right))
		y := max(-top, o.y)
		page := o.page
		// Objects may overlap one another. Preserve their anchors, moving to
		// another page only when the object itself extends past the paper.
		if y+lower > bottom+1e-8 {
			page++
			y = max(-top, min(s.MarginTop, flowBottom-lower))
		}
		if page > maxDocumentCharacters {
			return out, fmt.Errorf("页面数量过多，请拆分文档")
		}
		out.positions = append(out.positions, ObjectPosition{ID: o.id, Page: page, X: x, Y: y, Source: RegionBounds{o.x, o.y, frameW, frameH}, SourcePage: o.page, Rotation: o.rotation})
		if o.image >= 0 {
			out.images[o.image].Page, out.images[o.image].X, out.images[o.image].Y = page, x, y
		} else {
			// Source height is the configured minimum, not automatic text growth.
			out.positions[len(out.positions)-1].Source.Height = tableOffsets(tables[o.table].RowHeights)[len(tables[o.table].RowHeights)]
			out.tables[o.table].Page, out.tables[o.table].X, out.tables[o.table].Y = page, x, y
			out.renders[o.table].X, out.renders[o.table].Y = x, y
		}
	}
	return out, nil
}
