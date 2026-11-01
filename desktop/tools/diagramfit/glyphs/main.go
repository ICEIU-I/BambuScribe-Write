// Export only the existing handwriting glyphs required by an experiment.
package main

import (
	"bambuscribe/desktop/internal/engine"
	"encoding/json"
	"fmt"
	"os"
	"unicode"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: glyphs scene.json")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var scene struct {
		Labels []struct{ Text string }
	}
	if err := json.Unmarshal(raw, &scene); err != nil {
		panic(err)
	}
	font, err := engine.EmbeddedFont()
	if err != nil {
		panic(err)
	}
	glyphs := map[string]engine.Glyph{}
	for _, label := range scene.Labels {
		for _, char := range label.Text {
			if unicode.IsSpace(char) || char == '_' || char == '^' || char == '{' || char == '}' {
				continue
			}
			glyph, ok := font.Glyphs[char]
			if !ok {
				panic(fmt.Sprintf("missing glyph %q", char))
			}
			glyphs[string(char)] = glyph
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(glyphs); err != nil {
		panic(err)
	}
}
