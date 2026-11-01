package engine

import "math"

// LayoutWithImages uses the same occupied-space planner as tables and text.
func (f *Font) LayoutWithImages(text string, s LayoutSettings, assets []ImageAsset) (*Document, error) {
	return f.LayoutWithObjects(text, s, assets, nil)
}

// Smart figures reserve their complete frame; raster images use the rendered
// ink envelope so existing text in white margins retains its position.
// Trace once and reuse those exact strokes for export.
func (f *Font) prepareBodyOccupancy(plan *objectPlan, s LayoutSettings) error {
	plan.bodyOccupied = map[int][]RegionBounds{}
	plan.imageRenders = make([]imageRender, len(plan.images))
	positions := map[string]ObjectPosition{}
	for _, p := range plan.positions {
		positions[p.ID] = p
	}
	for i, asset := range plan.images {
		render, err := imageStrokes(f, asset, s)
		if err != nil {
			return err
		}
		plan.imageRenders[i] = render
		if asset.Diagram != nil {
			// Smart-imported figures reserve their complete frame. Their blank
			// margins belong to the figure too; following body lines must move
			// below it rather than remain inside that draggable frame.
			a := asset.Rotation * math.Pi / 180
			w := math.Abs(asset.Width*math.Cos(a)) + math.Abs(asset.Height*math.Sin(a))
			h := math.Abs(asset.Width*math.Sin(a)) + math.Abs(asset.Height*math.Cos(a))
			plan.bodyOccupied[asset.Page] = append(plan.bodyOccupied[asset.Page], RegionBounds{asset.X + (asset.Width-w)/2, asset.Y + (asset.Height-h)/2, w, h})
			p := positions[asset.ID]
			if p.Page > p.SourcePage {
				// An overflowing insertion leaves the original page after the
				// cursor empty, so later body text still follows the figure.
				top := max(0, p.Source.Y+(asset.Height-h)/2)
				plan.bodyOccupied[p.SourcePage] = append(plan.bodyOccupied[p.SourcePage], RegionBounds{0, top, s.PaperWidth, max(0, s.PaperHeight-top)})
			}
		} else if bounds, ok := imageInkBounds(render.Strokes); ok {
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
