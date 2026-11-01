package engine

import (
	"sort"
	"strconv"
)

// A body line or an entire object is one writing block. Sorting individual
// strokes would scramble character stroke order and split connected figures.
type writingBlockKey struct {
	object string
	row    float64
}

type writingBlock struct {
	key     writingBlockKey
	x, y    float64
	strokes []Stroke
	kinds   []string
}

type pageWritingOrder struct {
	rows   map[string]float64
	index  map[writingBlockKey]int
	blocks []writingBlock
}

func newPageWritingOrder(page Page) *pageWritingOrder {
	order := &pageWritingOrder{rows: map[string]float64{}, index: map[writingBlockKey]int{}}
	for _, cell := range page.Cells {
		order.rows[strconv.Itoa(cell.Start)] = cell.Y
	}
	return order
}

func (o *pageWritingOrder) add(layer InkLayer, strokes []Stroke) {
	bounds, ok := imageInkBounds(strokes)
	if !ok {
		return
	}
	key := writingBlockKey{object: layer.ObjectID}
	if layer.ObjectID == "" {
		key.row = o.rows[layer.Part]
	}
	i, exists := o.index[key]
	if !exists {
		i = len(o.blocks)
		o.index[key] = i
		o.blocks = append(o.blocks, writingBlock{key: key, x: bounds.X, y: bounds.Y})
	}
	block := &o.blocks[i]
	block.x, block.y = min(block.x, bounds.X), min(block.y, bounds.Y)
	block.strokes = append(block.strokes, strokes...)
	kind := ""
	if layer.ObjectID != "" && layer.Part == "grid" {
		kind = strokeKindTableGrid
	}
	for range strokes {
		block.kinds = append(block.kinds, kind)
	}
}

func (o *pageWritingOrder) strokes() ([]Stroke, []string) {
	// Paper Y increases downwards; sort before converting to machine Y.
	// Visible ink accounts for image whitespace, rotation and completed cuts.
	sort.SliceStable(o.blocks, func(i, j int) bool {
		a, b := o.blocks[i], o.blocks[j]
		if a.y != b.y {
			return a.y < b.y
		}
		if a.x != b.x {
			return a.x < b.x
		}
		// A deterministic tie keeps coincident objects independent of the
		// project array order, including after copy, undo and reopening.
		if a.key.object != b.key.object {
			return a.key.object < b.key.object
		}
		return a.key.row < b.key.row
	})
	out := []Stroke{}
	kinds := []string{}
	hasGrid := false
	for _, block := range o.blocks {
		out = append(out, block.strokes...)
		kinds = append(kinds, block.kinds...)
		for _, kind := range block.kinds {
			hasGrid = hasGrid || kind == strokeKindTableGrid
		}
	}
	if !hasGrid {
		kinds = nil
	}
	return out, kinds
}
