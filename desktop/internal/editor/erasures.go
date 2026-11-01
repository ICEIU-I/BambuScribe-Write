package editor

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"fmt"
	"reflect"
)

// Source edits generate new ink for that part; returning to an earlier string
// must not resurrect an old eraser action. Undo keeps its independent snapshot.
func discardReplacedInk(before project.Project, after *project.Project) {
	changed := map[string]map[string]bool{}
	for _, old := range before.Tables {
		var next *engine.TableAsset
		for i := range after.Tables {
			if after.Tables[i].ID == old.ID {
				next = &after.Tables[i]
				break
			}
		}
		if next == nil {
			changed[old.ID] = nil
			continue
		}
		parts := map[string]bool{}
		topology := len(old.Cells) != len(next.Cells) || len(old.ColumnWidths) != len(next.ColumnWidths) || len(old.RowHeights) != len(next.RowHeights) || old.Style != next.Style
		for i, c := range old.Cells {
			if i >= len(next.Cells) {
				break
			}
			n := next.Cells[i]
			if c.Row != n.Row || c.Col != n.Col || c.RowSpan != n.RowSpan || c.ColSpan != n.ColSpan || !reflect.DeepEqual(c.Borders, n.Borders) {
				topology = true
			}
			if c.Text != n.Text {
				parts[fmt.Sprintf("cell:%d:%d", c.Row, c.Col)] = true
			}
		}
		if topology {
			changed[old.ID] = nil
		} else if len(parts) > 0 {
			changed[old.ID] = parts
		}
	}
	for _, old := range before.Images {
		var next *engine.ImageAsset
		for i := range after.Images {
			if after.Images[i].ID == old.ID {
				next = &after.Images[i]
				break
			}
		}
		if next == nil || old.Data != next.Data || old.Mode != next.Mode || !reflect.DeepEqual(old.Diagram, next.Diagram) {
			changed[old.ID] = nil
			continue
		}
		parts := map[string]bool{}
		for _, r := range old.TextRegions {
			same := false
			for _, n := range next.TextRegions {
				if n.ID == r.ID && n.Text == r.Text && n.TextFormat == r.TextFormat && n.Mode == r.Mode {
					same = true
					break
				}
			}
			if !same {
				parts["region:"+r.ID] = true
			}
		}
		if len(parts) > 0 {
			changed[old.ID] = parts
		}
	}
	for i, e := range after.Erasures {
		if e.Cuts == nil {
			continue
		}
		cuts := []engine.InkCut{}
		for _, c := range *e.Cuts {
			parts, ok := changed[c.ObjectID]
			if !ok || parts != nil && !parts[c.Part] {
				cuts = append(cuts, c)
			}
		}
		after.Erasures[i].Cuts = &cuts
	}
}
