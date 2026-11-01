package engine

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
)

//go:embed fontdata/handwriting.json.gz
var fontData []byte

type Glyph struct {
	Width   float64
	Strokes []Stroke
}

type Font struct {
	ID     string
	Name   string
	Glyphs map[rune]Glyph
	Units  float64
}

var fontOnce sync.Once
var embeddedFont *Font
var embeddedError error

func EmbeddedFont() (*Font, error) {
	fontOnce.Do(func() {
		r, err := gzip.NewReader(bytes.NewReader(fontData))
		if err != nil {
			embeddedError = err
			return
		}
		defer r.Close()
		embeddedFont, embeddedError = LoadFont(r)
		if embeddedFont != nil {
			embeddedFont.ID, embeddedFont.Name = FontHandwriting, "教案专用"
		}
	})
	return embeddedFont, embeddedError
}

func LoadFont(r io.Reader) (*Font, error) {
	var raw map[string][]json.RawMessage
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, err
	}
	f := &Font{Glyphs: make(map[rune]Glyph), Units: 11}
	for char, value := range raw {
		key := []rune(char)
		if len(key) != 1 || len(value) != 3 {
			return nil, fmt.Errorf("无效字形：%s", char)
		}
		var width float64
		var xx, yy [][]float64
		if json.Unmarshal(value[0], &width) != nil || json.Unmarshal(value[1], &xx) != nil || json.Unmarshal(value[2], &yy) != nil || len(xx) != len(yy) || !finite(width) || width < 0 {
			return nil, fmt.Errorf("无效字形坐标：%s", char)
		}
		g := Glyph{Width: width}
		for i, xs := range xx {
			if len(xs) != len(yy[i]) || len(xs) < 2 {
				return nil, fmt.Errorf("无效笔画：%s", char)
			}
			stroke := make(Stroke, len(xs))
			for j, x := range xs {
				y := yy[i][j]
				if !finite(x) || !finite(y) || x < 0 || y < 0 || x > f.Units || y > f.Units {
					return nil, fmt.Errorf("字形超出坐标范围：%s", char)
				}
				stroke[j] = Point{x, y}
			}
			g.Strokes = append(g.Strokes, stroke)
		}
		f.Glyphs[key[0]] = g
	}
	addPunctuation(f)
	addSupplementalSymbols(f)
	return f, nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func addPunctuation(f *Font) {
	put := func(chars string, width float64, strokes ...Stroke) {
		for _, r := range chars {
			if _, exists := f.Glyphs[r]; !exists {
				f.Glyphs[r] = Glyph{width, strokes}
			}
		}
	}
	dot := Stroke{{0, 7.5}, {0.22, 7.5}}
	put(".", 0.4, dot)
	put("。", 1.6, Stroke{{0, 7.4}, {0.4, 6.9}, {1, 6.9}, {1.5, 7.4}, {1, 8}, {0.4, 8}, {0, 7.4}})
	put("，", 1.1, Stroke{{0.8, 7.1}, {1, 7.6}, {0, 8.7}})
	put("、", 1.6, Stroke{{0, 6.6}, {1.6, 8.1}})
	put("!！", 0.7, Stroke{{0.3, 2}, {0.3, 6}}, dot)
	put("?？", 3.1, Stroke{{0, 3}, {0.6, 2}, {2, 2}, {3, 2.8}, {2.8, 4}, {1.4, 5}, {1.4, 6}}, Stroke{{1.4, 7.5}, {1.6, 7.5}})
	put(":：", 0.4, Stroke{{0, 4.2}, {0.2, 4.2}}, dot)
	put(";；", 1, Stroke{{0.6, 4.2}, {0.8, 4.2}}, Stroke{{0.8, 7.1}, {1, 7.6}, {0, 8.7}})
	put("(（", 2, Stroke{{2, 1.4}, {0.7, 3}, {0, 5}, {0.7, 7.5}, {2, 9}})
	put(")）", 2, Stroke{{0, 1.4}, {1.3, 3}, {2, 5}, {1.3, 7.5}, {0, 9}})
	put("[【", 2, Stroke{{2, 1.5}, {0, 1.5}, {0, 9}, {2, 9}})
	put("]】", 2, Stroke{{0, 1.5}, {2, 1.5}, {2, 9}, {0, 9}})
	put("{", 2, Stroke{{2, 1.5}, {1, 2}, {1, 4}, {0, 5}, {1, 6}, {1, 8}, {2, 9}})
	put("}", 2, Stroke{{0, 1.5}, {1, 2}, {1, 4}, {2, 5}, {1, 6}, {1, 8}, {0, 9}})
	put("‘", 1, Stroke{{1, 1.5}, {0, 2.5}, {0.4, 3}})
	put("’", 1, Stroke{{0.6, 1.5}, {1, 2}, {0, 3}})
	put("“", 2.5, Stroke{{1, 1.5}, {0, 2.5}, {0.4, 3}}, Stroke{{2.5, 1.5}, {1.5, 2.5}, {1.9, 3}})
	put("”", 2.5, Stroke{{0.6, 1.5}, {1, 2}, {0, 3}}, Stroke{{2.1, 1.5}, {2.5, 2}, {1.5, 3}})
	put("《", 3, Stroke{{1.5, 2}, {0, 5}, {1.5, 8}}, Stroke{{3, 2}, {1.5, 5}, {3, 8}})
	put("》", 3, Stroke{{0, 2}, {1.5, 5}, {0, 8}}, Stroke{{1.5, 2}, {3, 5}, {1.5, 8}})
	put("〈", 1.5, Stroke{{1.5, 2}, {0, 5}, {1.5, 8}})
	put("〉", 1.5, Stroke{{0, 2}, {1.5, 5}, {0, 8}})
	put("…", 5, Stroke{{0, 6}, {0.2, 6}}, Stroke{{2.4, 6}, {2.6, 6}}, Stroke{{4.8, 6}, {5, 6}})
	put("·", 0.4, Stroke{{0, 5}, {0.3, 5}})
	put("$¥￥＄", 4, Stroke{{0, 2}, {2, 4}, {4, 2}}, Stroke{{0, 4.7}, {4, 4.7}}, Stroke{{0, 6}, {4, 6}}, Stroke{{2, 4}, {2, 8}})
	put("×", 4, Stroke{{0, 3}, {4, 7}}, Stroke{{0, 7}, {4, 3}})
	put("÷", 4, Stroke{{0, 5}, {4, 5}}, Stroke{{1.9, 2.7}, {2.1, 2.7}}, Stroke{{1.9, 7.3}, {2.1, 7.3}})
	put("±", 4, Stroke{{0, 4}, {4, 4}}, Stroke{{2, 2}, {2, 6}}, Stroke{{0, 8}, {4, 8}})
	put("°", 1.4, Stroke{{0, 2}, {0.4, 1.5}, {1, 1.5}, {1.4, 2}, {1, 2.6}, {0.4, 2.6}, {0, 2}})
	put("μµ", 4, Stroke{{0.5, 3.5}, {0, 9}}, Stroke{{0.5, 3.5}, {0.4, 6}, {1, 7.2}, {2.2, 7}, {3.3, 3.5}, {2.8, 6.8}, {3.3, 7.3}, {4, 6.8}})
	put("℃", 5.5, Stroke{{0, 2}, {0.4, 1.5}, {1, 1.5}, {1.4, 2}, {1, 2.6}, {0.4, 2.6}, {0, 2}}, Stroke{{5.5, 3}, {4.6, 2.4}, {3.2, 2.6}, {2.4, 4.1}, {2.4, 6}, {3.3, 7.5}, {4.7, 7.8}, {5.5, 7.2}})
}
