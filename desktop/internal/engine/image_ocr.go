package engine

import (
	"math"
	"unicode"
)

// A symbol sharing a line with a label is sometimes recognised as a character:
// the light-source symbol came back as 必 inside a "光源 必" box, so the
// replacement erased the symbol and drew the wrong word over it. Nothing in the
// OCR result separates the two, so every character is checked against the ink
// it claims to cover, and the ones that do not match keep their line art.

// glyphAgreement is the least share of a character's ink that its rendered
// glyph must cover for the reading to be trusted. Measured on a textbook
// schematic: characters read correctly scored 0.53-0.92, the light-source
// symbol misread as a character scored 0.43. The two mistakes do not cost the
// same — rejecting a correct character only leaves it printed instead of
// handwritten, while accepting a wrong one draws a garble over the artwork — so
// the threshold sits nearer the misread than the midpoint.
const glyphAgreement = .5

// Cells narrower than this cannot be judged: a couple of pixels of stroke,
// smeared by the comparison's tolerance, say nothing about which character they
// are. They are trusted rather than guessed at.
const minAgreementCell = 6

// ocrText is a run of adjacent characters from one box that are safe to
// replace. box places the handwriting; source is the ink it covers.
type ocrText struct {
	text   string
	box    OCRBox
	source RegionBounds
}

type textCell struct {
	from, to int // image pixel columns
	rune     int // index into the original rune slice, keeping spaces in runs
}

// verifiedOcrText reports the runs of characters in a box whose glyphs cover
// the ink beneath them. Symbols the recogniser mistook for characters fall out
// of every run, so their pixels survive as line art.
//
// The reading is only judged when the blank columns separate exactly as many
// cells as there are characters. A box the splitter could not take apart (a
// tight "30", or ink heavy enough to close the gaps) is handed back whole,
// which is what every box did before.
func verifiedOcrText(f *Font, box OCRBox, split, ink []bool, width, height int, asset ImageAsset) []ocrText {
	parent := box
	runes := []rune(box.Text)
	letters := 0
	for _, r := range runes {
		if !unicode.IsSpace(r) {
			letters++
		}
	}
	// Split against the ink as printed: a rule crossing the label would read as
	// a break between characters. Judge against the ink left after rules are
	// removed, or that rule's band is ink no glyph could ever cover.
	cells := splitTextCells(split, width, height, box)
	if letters == 0 || len(cells) != letters {
		return wholeBoxText(box)
	}
	next := 0
	for i, r := range runes {
		if unicode.IsSpace(r) {
			continue
		}
		cells[next].rune = i
		next++
	}
	keep := make([]bool, len(cells))
	rejected := false
	for i, cell := range cells {
		keep[i] = cellAgreement(f, box, cell, runes[cell.rune], ink, width, height, asset) >= glyphAgreement
		rejected = rejected || !keep[i]
	}
	if !rejected {
		return wholeBoxText(box)
	}
	out := []ocrText{}
	tight := func(from, to int) RegionBounds {
		return RegionBounds{float64(from) / float64(width), box.Y, float64(to-from) / float64(width), box.H}
	}
	for i := 0; i < len(cells); {
		if !keep[i] {
			i++
			continue
		}
		j := i
		for j+1 < len(cells) && keep[j+1] {
			j++
		}
		// Any spaces between the surviving characters stay in the run.
		run := ocrText{text: string(runes[cells[i].rune : cells[j].rune+1]), source: tight(cells[i].from, cells[j].to)}
		run.box = runBox(run.text, run.source, parent)
		out = append(out, run)
		i = j + 1
	}
	return out
}

// wholeBoxText is the unjudged reading: one run covering the box as recognised.
func wholeBoxText(box OCRBox) []ocrText {
	return []ocrText{{text: box.Text, box: box, source: RegionBounds{box.X, box.Y, box.W, box.H}}}
}

// A fragment owns only its source interval. Giving a lone hyphen the parent
// line's width enlarged a 17 px mark to 166 px across neighbouring characters.
func runBox(text string, source RegionBounds, _ OCRBox) OCRBox {
	return OCRBox{Text: text, X: source.X, Y: source.Y, W: source.Width, H: source.Height}
}

// splitTextCells cuts a recognised box at the blank columns between glyphs.
func splitTextCells(mask []bool, width, height int, box OCRBox) []textCell {
	from := max(0, int(math.Floor(box.X*float64(width))))
	to := min(width-1, int(math.Ceil((box.X+box.W)*float64(width))))
	top := max(0, int(math.Floor(box.Y*float64(height))))
	bottom := min(height-1, int(math.Ceil((box.Y+box.H)*float64(height))))
	if to <= from || bottom < top {
		return nil
	}
	cells := []textCell{}
	start := -1
	for x := from; x <= to+1; x++ {
		inked := false
		if x <= to {
			for y := top; y <= bottom; y++ {
				if mask[y*width+x] {
					inked = true
					break
				}
			}
		}
		if inked && start < 0 {
			start = x
		}
		if !inked && start >= 0 {
			cells = append(cells, textCell{from: start, to: x})
			start = -1
		}
	}
	return cells
}

// cellAgreement is the share of a cell's ink that the character's rendered
// glyph covers. Ink beside the glyph, as a symbol would leave, is what pulls
// the score down.
func cellAgreement(f *Font, box OCRBox, cell textCell, r rune, ink []bool, width, height int, asset ImageAsset) float64 {
	box.X = float64(cell.from) / float64(width)
	box.W = float64(cell.to-cell.from) / float64(width)
	box.Text = string(r)
	top := max(0, int(math.Floor(box.Y*float64(height))))
	bottom := min(height-1, int(math.Ceil((box.Y+box.H)*float64(height))))
	w, h := cell.to-cell.from, bottom-top+1
	if w < minAgreementCell || h < 1 {
		return 1
	}
	original := make([]bool, w*h)
	total := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			original[y*w+x] = ink[(top+y)*width+cell.from+x]
			if original[y*w+x] {
				total++
			}
		}
	}
	if total == 0 {
		return 1
	}
	strokes := handwritingTextStrokes(f, box, asset, DefaultLayout(), width, height)
	if len(strokes) == 0 {
		return 0
	}
	rendered := make([]bool, w*h)
	mark := func(x, y int) {
		if x >= 0 && x < w && y >= 0 && y < h {
			rendered[y*w+x] = true
		}
	}
	for _, s := range strokes {
		for i, p := range s {
			// Stroke coordinates are paper millimetres; undo the placement and
			// the raster scale to compare them with the ink they came from.
			x := (p.X-asset.X)/asset.Width*float64(width-1) - float64(cell.from)
			y := (p.Y-asset.Y)/asset.Height*float64(height-1) - float64(top)
			if i == 0 {
				mark(int(math.Round(x)), int(math.Round(y)))
				continue
			}
			q := s[i-1]
			px := (q.X-asset.X)/asset.Width*float64(width-1) - float64(cell.from)
			py := (q.Y-asset.Y)/asset.Height*float64(height-1) - float64(top)
			steps := int(math.Max(math.Abs(x-px), math.Abs(y-py))) + 1
			for k := 0; k <= steps; k++ {
				t := float64(k) / float64(steps)
				mark(int(math.Round(px+t*(x-px))), int(math.Round(py+t*(y-py))))
			}
		}
	}
	if !anyInk(rendered) {
		return 0
	}
	// Share of the cell's own ink that a pen path passes within a pixel of.
	covered := 0
	drawn := dilateMask(rendered, w, h, 1)
	for i, inked := range original {
		if inked && drawn[i] {
			covered++
		}
	}
	return float64(covered) / float64(total)
}

func anyInk(mask []bool) bool {
	for _, v := range mask {
		if v {
			return true
		}
	}
	return false
}

func dilateMask(mask []bool, width, height, radius int) []bool {
	out := make([]bool, len(mask))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if !mask[y*width+x] {
				continue
			}
			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						out[ny*width+nx] = true
					}
				}
			}
		}
	}
	return out
}
