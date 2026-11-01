package engine

import "math"

// LayoutWithImages uses the same occupied-space planner as tables and text.
func (f *Font) LayoutWithImages(text string, s LayoutSettings, assets []ImageAsset) (*Document, error) {
	return f.LayoutWithObjects(text, s, assets, nil)
}

// Image frames still participate in object placement. Body text clears the
// rendered ink envelope, allowing existing text in the raster's white margins
// to retain its position. Trace once and reuse those exact strokes for export.
func (f *Font) prepareBodyOccupancy(plan *objectPlan, s LayoutSettings) error {
	plan.bodyOccupied = map[int][]RegionBounds{}
	plan.imageRenders = make([]imageRender, len(plan.images))
	for i, asset := range plan.images {
		render, err := imageStrokes(f, asset, s)
		if err != nil {
			return err
		}
		plan.imageRenders[i] = render
		if bounds, ok := imageInkBounds(render.Strokes); ok {
			plan.bodyOccupied[asset.Page] = append(plan.bodyOccupied[asset.Page], bounds)
		}
	}
	for i, table := range plan.tables {
		r := plan.renders[i]
		plan.bodyOccupied[table.Page] = append(plan.bodyOccupied[table.Page], RegionBounds{r.X, r.Y, r.Width, r.Height})
	}
	return nil
}

// A horizontal/vertical line still occupies space. Unlike glyph fitting,
// content exclusion must retain envelopes with zero width or height.
func imageInkBounds(strokes []Stroke) (RegionBounds, bool) {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, stroke := range strokes {
		for _, p := range stroke {
			x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
		}
	}
	if !finite(x0) {
		return RegionBounds{}, false
	}
	return RegionBounds{x0, y0, x1 - x0, y1 - y0}, true
}

func appendImages(doc *Document, assets []ImageAsset, renders []imageRender) {
	for i, asset := range assets {
		for len(doc.Pages) <= asset.Page {
			doc.Pages = append(doc.Pages, blankLayoutPage(doc.Settings, len(doc.Pages)))
		}
		render := renders[i]
		page := &doc.Pages[asset.Page]
		page.Strokes = append(page.Strokes, render.Strokes...)
		page.Warnings = append(page.Warnings, render.Warnings...)
		page.Images = append(page.Images, ImagePlacement{ID: asset.ID, X: asset.X, Y: asset.Y, Width: asset.Width, Height: asset.Height, Rotation: asset.Rotation, Mode: asset.Mode, Strokes: render.Strokes, GraphicStrokes: render.GraphicStrokes, SourceStrokes: render.SourceStrokes, TextRegions: render.TextRegions})
	}
}
