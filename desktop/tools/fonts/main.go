// Export a complete built-in stroke font in the original [width, X, Y] format.
package main

import (
	"bambuscribe/desktop/internal/engine"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	id := flag.String("font", engine.FontDC, "built-in font ID")
	output := flag.String("output", "", "output JSON path")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "请指定 --output 字体.json")
		os.Exit(1)
	}
	f, err := engine.BuiltinFont(*id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw := map[string]any{}
	for r, g := range f.Glyphs {
		xx, yy := [][]float64{}, [][]float64{}
		for _, stroke := range g.Strokes {
			xs, ys := []float64{}, []float64{}
			for _, p := range stroke {
				xs, ys = append(xs, p.X), append(ys, p.Y)
			}
			xx, yy = append(xx, xs), append(yy, ys)
		}
		raw[string(r)] = []any{g.Width, xx, yy}
	}
	b, err := json.Marshal(raw)
	if err == nil {
		err = os.WriteFile(*output, b, 0644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s：%d 字形，已保存 %s\n", f.Name, len(f.Glyphs), *output)
}
