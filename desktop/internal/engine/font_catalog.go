package engine

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"sync"
)

const (
	FontHandwriting = "handwriting"
	FontDC          = "dc"
)

//go:embed fontdata/dc.json.gz
var dcFontData []byte

var dcFontOnce sync.Once
var dcFont *Font
var dcFontError error

// BuiltinFont returns immutable, independently supplemented stroke fonts.
// An absent ID keeps old drafts and projects on their original font.
func BuiltinFont(id string) (*Font, error) {
	switch id {
	case "", FontHandwriting:
		return EmbeddedFont()
	case FontDC:
		dcFontOnce.Do(func() {
			r, err := gzip.NewReader(bytes.NewReader(dcFontData))
			if err != nil {
				dcFontError = err
				return
			}
			defer r.Close()
			dcFont, dcFontError = LoadFont(r)
			if dcFont != nil {
				dcFont.ID, dcFont.Name = FontDC, "DC体"
			}
		})
		return dcFont, dcFontError
	default:
		return nil, fmt.Errorf("未知字体 %q，请在排版设置中重新选择字体", id)
	}
}

// ForLayout selects a named font before any text or object measurement.
// Callers supplying their own Font retain it when no font ID is specified.
func (f *Font) ForLayout(s LayoutSettings) (*Font, error) {
	if s.FontID == "" || s.FontID == f.ID {
		return f, nil
	}
	return BuiltinFont(s.FontID)
}
