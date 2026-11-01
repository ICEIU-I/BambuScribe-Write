package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

// PageErasure retains gesture bounds for history. Cuts identify the source
// vector portions deleted at completion; bounds are only used for migration.
type PageErasure struct {
	ID     string  `json:"id"`
	Page   int     `json:"page"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Path   []Point `json:"path,omitempty"`
	Radius float64 `json:"radius,omitempty"`
	// Legacy object scope, used only when converting records without Cuts.
	ObjectIDs *[]string `json:"objectIds,omitempty"`
	// Non-nil means this gesture is materialized: paper bounds are history only.
	Cuts *[]InkCut `json:"cuts,omitempty"`
}

func ValidateErasures(erasures []PageErasure) error {
	if len(erasures) > 2000 {
		return fmt.Errorf("擦除记录过多，请拆分文档")
	}
	seen := map[string]bool{}
	total := 0
	cutTotal := 0
	valid := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1000 }
	for _, e := range erasures {
		if e.ID == "" || seen[e.ID] || e.Page < 0 || e.Page > maxDocumentCharacters {
			return fmt.Errorf("擦除记录标识或页码无效，请撤销后重试")
		}
		seen[e.ID] = true
		if err := validateInkCuts(e.Cuts, &cutTotal); err != nil {
			return err
		}
		if e.ObjectIDs != nil {
			ids := map[string]bool{}
			if len(*e.ObjectIDs) > 200 {
				return fmt.Errorf("擦除对象过多，请拆分文档")
			}
			for _, id := range *e.ObjectIDs {
				if id == "" || len(id) > 1000 || ids[id] {
					return fmt.Errorf("擦除对象标识无效，请撤销后重试")
				}
				ids[id] = true
			}
		}
		if !valid(e.X) || !valid(e.Y) || !valid(e.Width) || !valid(e.Height) {
			return fmt.Errorf("擦除范围无效，请撤销后重试")
		}
		if len(e.Path) == 0 {
			if e.Width <= 0 || e.Height <= 0 {
				return fmt.Errorf("擦除范围不能为空")
			}
		} else {
			total += len(e.Path)
			if len(e.Path) > 20000 || total > 200000 || !valid(e.Radius) || e.Radius <= 0 || e.Radius > 20 {
				return fmt.Errorf("自由擦除轨迹或粗细无效，请缩短轨迹后重试")
			}
			for _, p := range e.Path {
				if !valid(p.X) || !valid(p.Y) {
					return fmt.Errorf("自由擦除坐标无效，请撤销后重试")
				}
			}
		}
	}
	return nil
}

// Keep editable source geometry intact. Preview and export remove the same
// stored vector portions; completed gestures never mask paper coordinates.
func (f *Font) LayoutWithErasures(text string, s LayoutSettings, images []ImageAsset, tables []TableAsset, erasures []PageErasure) (*Document, error) {
	if err := ValidateErasures(erasures); err != nil {
		return nil, err
	}
	doc, err := f.LayoutWithObjects(text, s, images, tables)
	if err != nil || len(erasures) == 0 {
		return doc, err
	}
	// Migrate old projects once against this preview; clients persist these cuts.
	erasures = append([]PageErasure(nil), erasures...)
	for i, e := range erasures {
		if e.Cuts != nil {
			continue
		}
		cuts := []InkCut{}
		if e.Page < len(doc.Pages) {
			cuts = captureInk(e, doc.Pages[e.Page].Ink)
		}
		erasures[i].Cuts = &cuts
		ids := []string{}
		for _, c := range cuts {
			if c.ObjectID != "" && !containsID(ids, c.ObjectID) {
				ids = append(ids, c.ObjectID)
			}
		}
		erasures[i].ObjectIDs = &ids
	}
	doc.Erasures = erasures
	for i := range doc.Pages {
		doc.Pages[i].Strokes, doc.Pages[i].StrokeKinds = erasedPageInk(doc.Pages[i], erasures)
	}

	encoded, _ := json.Marshal(struct {
		Base     string
		Erasures []PageErasure
		Version  string
	}{doc.ID, erasures, "paper-eraser-v3-once"})
	h := sha256.Sum256(encoded)
	doc.ID = hex.EncodeToString(h[:12])
	return doc, nil
}

func erasePageStrokes(strokes []Stroke, erasures []PageErasure) []Stroke {
	regions := make([]ImageTextRegion, 0, len(erasures))
	for _, e := range erasures {
		regions = append(regions, ImageTextRegion{Source: "manual", Mode: "erase", X: e.X, Y: e.Y, Width: e.Width, Height: e.Height, ErasePath: e.Path, EraseRadius: e.Radius})
	}
	// An identity frame lets the existing segment/capsule clipper use mm.
	return eraseImageStrokes(strokes, ImageAsset{Width: 1, Height: 1}, regions)
}
