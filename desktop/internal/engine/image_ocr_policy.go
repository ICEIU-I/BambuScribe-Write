package engine

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

const imageOCRVersion = 4

// General-purpose OCR does not encode mathematical layout. Only replace plain
// text with strong recognition evidence; everything else keeps its source ink.
// In particular, confidence is not a probability and cannot validate a formula.
func plainOCRLabel(box OCRBox) bool {
	if !finite(box.Confidence) || box.Confidence < .8 {
		return false
	}
	letters := 0
	for _, r := range box.Text {
		switch {
		case unicode.IsLetter(r), r >= '0' && r <= '9':
			letters++
		case unicode.IsSpace(r), strings.ContainsRune("，。！？、：；,.!?:;", r):
		default:
			return false
		}
	}
	return letters > 0
}

func clippedOCRBox(box OCRBox) (OCRBox, bool) {
	if !finite(box.X) || !finite(box.Y) || !finite(box.W) || !finite(box.H) || box.W <= 0 || box.H <= 0 {
		return OCRBox{}, false
	}
	right, bottom := math.Min(1, box.X+box.W), math.Min(1, box.Y+box.H)
	box.X, box.Y = math.Max(0, box.X), math.Max(0, box.Y)
	box.W, box.H = right-box.X, bottom-box.Y
	return box, box.W > 0 && box.H > 0
}

// Previously editable OCR objects may have been moved or explicitly kept by
// the user. Refresh only unchanged automatic replacements, including the old
// expanded fragment widths; preserve all manual operations and their order.
func staleOCRRegion(r ImageTextRegion) bool {
	if r.Source != "ocr" || r.OCRVersion >= imageOCRVersion || (r.Mode != "handwriting" && r.Mode != "keep") || r.SourceBounds == nil {
		return false
	}
	b := r.SourceBounds
	return math.Abs(r.X-b.X) < 1e-9 && math.Abs(r.Y-b.Y) < 1e-9 &&
		math.Abs(r.Height-b.Height) < 1e-9 && r.Width >= b.Width-1e-9
}

func needsImageOCR(regions []ImageTextRegion) bool {
	if regions == nil {
		return true
	}
	for _, r := range regions {
		if staleOCRRegion(r) {
			return true
		}
	}
	return false
}

func resolveImageOCR(f *Font, existing []ImageTextRegion, boxes []OCRBox, original, ink []bool, width, height int, asset ImageAsset) []ImageTextRegion {
	retained := make([]ImageTextRegion, 0, len(existing))
	for _, r := range existing {
		if !staleOCRRegion(r) {
			retained = append(retained, r)
		}
	}
	// If recognition is unavailable (including non-macOS), restore the ink of
	// stale replacements instead of continuing to draw cached garbled strings.
	if len(boxes) == 0 {
		for _, r := range existing {
			if staleOCRRegion(r) {
				b := r.SourceBounds
				boxes = append(boxes, OCRBox{Text: r.Text, X: b.X, Y: b.Y, W: b.Width, H: b.Height})
			}
		}
	}
	regions := make([]ImageTextRegion, 0, len(boxes)+len(retained))
	used := map[string]bool{}
	for _, r := range retained {
		used[r.ID] = true
	}
	next := 0
	for _, candidate := range boxes {
		box, ok := clippedOCRBox(candidate)
		if !ok || coveredByRetainedOCR(box, retained) {
			continue
		}
		source := RegionBounds{box.X, box.Y, box.W, box.H}
		mode := "keep"
		if box.Backend == "local-hybrid" {
			if box.Replace {
				mode = "handwriting"
			}
		} else if plainOCRLabel(box) && ocrCellsMatchText(box, original, width, height) {
			runs := verifiedOcrText(f, box, original, ink, width, height, asset)
			// No mixtures of a few handwritten characters and unverified print.
			if len(runs) == 1 && runs[0].text == box.Text {
				mode = "handwriting"
			}
		}
		var id string
		for {
			next++
			id = fmt.Sprintf("ocr-v%d-%d", imageOCRVersion, next)
			if !used[id] {
				used[id] = true
				break
			}
		}
		regions = append(regions, ImageTextRegion{
			ID: id, Text: box.Text, X: box.X, Y: box.Y, Width: box.W, Height: box.H,
			Mode: mode, Source: "ocr", Confidence: box.Confidence, TextFormat: box.TextFormat,
			OCRVersion: imageOCRVersion, SourceBounds: &source,
		})
	}
	return append(regions, retained...)
}

func ocrCellsMatchText(box OCRBox, ink []bool, width, height int) bool {
	letters := 0
	for _, r := range box.Text {
		if !unicode.IsSpace(r) {
			letters++
		}
	}
	return letters > 0 && len(splitTextCells(ink, width, height, box)) == letters
}

func coveredByRetainedOCR(box OCRBox, regions []ImageTextRegion) bool {
	for _, r := range regions {
		if r.Source == "manual" {
			continue
		}
		b := RegionBounds{r.X, r.Y, r.Width, r.Height}
		if r.SourceBounds != nil {
			b = *r.SourceBounds
		}
		w := math.Max(0, math.Min(box.X+box.W, b.X+b.Width)-math.Max(box.X, b.X))
		h := math.Max(0, math.Min(box.Y+box.H, b.Y+b.Height)-math.Max(box.Y, b.Y))
		if w*h > .5*math.Min(box.W*box.H, b.Width*b.Height) {
			return true
		}
	}
	return false
}
