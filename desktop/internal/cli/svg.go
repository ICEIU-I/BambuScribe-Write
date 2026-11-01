package cli

import (
	"bambuscribe/desktop/internal/engine"
	"fmt"
	"strings"
)

// SVG uses the final vectors consumed by G-code, in paper millimeters. Editor
// handles and nonprinting paper guides are deliberately absent from the file.
func svgPage(d *engine.Document, index int) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%gmm" height="%gmm" viewBox="0 0 %g %g"><rect width="100%%" height="100%%" fill="white"/><g fill="none" stroke="#1b1a2e" stroke-width="0.26" stroke-linecap="round" stroke-linejoin="round">`, d.Settings.PaperWidth, d.Settings.PaperHeight, d.Settings.PaperWidth, d.Settings.PaperHeight)
	for _, s := range d.Pages[index].Strokes {
		b.WriteString(`<path d="`)
		for i, p := range s {
			command := "L"
			if i == 0 {
				command = "M"
			}
			fmt.Fprintf(&b, "%s%g,%g ", command, p.X, p.Y)
		}
		b.WriteString(`"/>`)
	}
	b.WriteString(`</g><g fill="none" stroke="red" stroke-width="0.3">`)
	for _, m := range d.Pages[index].Missing {
		fmt.Fprintf(&b, `<rect x="%g" y="%g" width="%g" height="%g"/>`, m.X, m.Y, m.Size, m.Size)
	}
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}
