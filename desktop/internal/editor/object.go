package editor

import (
	"bambuscribe/desktop/internal/project"
	"fmt"
	"github.com/google/uuid"
	"math"
)

type objectParams struct {
	ID        string   `json:"id"`
	NewID     string   `json:"newId"`
	Page      *int     `json:"page"`
	X         *float64 `json:"x"`
	Y         *float64 `json:"y"`
	Width     *float64 `json:"width"`
	Height    *float64 `json:"height"`
	ScaleText bool     `json:"scaleText"`
}

func editObject(p *project.Project, op Operation) (string, error) {
	var a objectParams
	if err := Decode(op.Params, &a); err != nil {
		return "", err
	}
	if a.Page != nil && (*a.Page < 1 || *a.Page > 20001) {
		return "", fmt.Errorf("页码须为 1–20001")
	}
	if op.Tool == "object.copy" {
		if a.NewID == "" {
			a.NewID = "object-" + uuid.NewString()
		}
		if err := uniqueID(p, a.NewID); err != nil {
			return "", err
		}
	}
	for i, t := range p.Tables {
		if t.ID == a.ID {
			if op.Tool == "object.delete" {
				p.Tables = append(p.Tables[:i], p.Tables[i+1:]...)
				return a.ID, nil
			}
			if op.Tool == "object.copy" {
				t.ID = a.NewID
				t.GroupID = ""
				t.X += 2
				t.Y += 2
			}
			if a.Page != nil {
				t.Page = *a.Page - 1
			}
			if a.X != nil {
				t.X = *a.X
			}
			if a.Y != nil {
				t.Y = *a.Y
			}
			if op.Tool == "object.resize" {
				if a.Width == nil || a.Height == nil {
					return "", fmt.Errorf("缩放须同时指定 width 和 height，单位 mm")
				}
				sx, sy := *a.Width/sum(t.ColumnWidths), *a.Height/sum(t.RowHeights)
				for j := range t.ColumnWidths {
					t.ColumnWidths[j] *= sx
				}
				for j := range t.RowHeights {
					t.RowHeights[j] *= sy
				}
				if a.ScaleText {
					t.FontSize = math.Max(1, math.Min(40, t.FontSize*math.Min(sx, sy)))
				}
			}
			if op.Tool == "object.copy" {
				p.Tables = append(p.Tables, t)
			} else {
				p.Tables[i] = t
			}
			return t.ID, nil
		}
	}
	for i, t := range p.Images {
		if t.ID == a.ID {
			if op.Tool == "object.delete" {
				p.Images = append(p.Images[:i], p.Images[i+1:]...)
				return a.ID, nil
			}
			if op.Tool == "object.copy" {
				t.ID = a.NewID
				t.X += 2
				t.Y += 2
			}
			if a.Page != nil {
				t.Page = *a.Page - 1
			}
			if a.X != nil {
				t.X = *a.X
			}
			if a.Y != nil {
				t.Y = *a.Y
			}
			if op.Tool == "object.resize" {
				if a.Width == nil || a.Height == nil {
					return "", fmt.Errorf("缩放须同时指定 width 和 height，单位 mm")
				}
				t.Width = *a.Width
				t.Height = *a.Height
			}
			if op.Tool == "object.copy" {
				p.Images = append(p.Images, t)
			} else {
				p.Images[i] = t
			}
			return t.ID, nil
		}
	}
	return "", fmt.Errorf("找不到对象 %q，请用 inspect 查看对象 ID", a.ID)
}
func sum(a []float64) float64 {
	var n float64
	for _, v := range a {
		n += v
	}
	return n
}
