// Test-only stdin bridge for browser tests against the real handwriting engine.
package main

import (
	"bambuscribe/desktop/internal/calculation"
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"bambuscribe/desktop/internal/tableimport"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	var input struct {
		Calculation *calculation.Plan `json:"calculation"`
		Import      *struct {
			Project    project.Project        `json:"project"`
			Extraction tableimport.Extraction `json:"extraction"`
			Placement  tableimport.Placement  `json:"placement"`
		} `json:"import"`
		Text     string                `json:"text"`
		Settings engine.LayoutSettings `json:"settings"`
		Images   []engine.ImageAsset   `json:"images"`
		Tables   []engine.TableAsset   `json:"tables"`
		Erasures []engine.PageErasure  `json:"erasures"`
		Export   bool                  `json:"export"`
		Page     int                   `json:"page"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fail(err)
		return
	}
	font, err := engine.EmbeddedFont()
	if err != nil {
		fail(err)
		return
	}
	if input.Calculation != nil {
		result, err := input.Calculation.Calculate()
		if err != nil {
			fail(err)
			return
		}
		doc, err := font.Layout(result.Text, input.Settings)
		if err != nil {
			fail(err)
			return
		}
		json.NewEncoder(os.Stdout).Encode(struct {
			calculation.Result
			Document *engine.Document `json:"document"`
		}{result, doc})
		return
	}
	if input.Import != nil {
		plan, err := tableimport.Build(font, input.Import.Project, input.Import.Extraction, input.Import.Placement)
		if err != nil {
			fail(err)
			return
		}
		json.NewEncoder(os.Stdout).Encode(plan)
		return
	}
	doc, err := font.LayoutWithErasures(input.Text, input.Settings, input.Images, input.Tables, input.Erasures)
	if err != nil {
		fail(err)
		return
	}
	if input.Export {
		if input.Page < 0 || input.Page >= len(doc.Pages) {
			fail(fmt.Errorf("invalid export page"))
			return
		}
		gcode, err := engine.GenerateGCode(doc.Pages[input.Page], doc.Settings, engine.DefaultWriting())
		if err != nil {
			fail(err)
			return
		}
		json.NewEncoder(os.Stdout).Encode(map[string]any{"document": doc, "gcode": gcode})
		return
	}
	json.NewEncoder(os.Stdout).Encode(doc)
}
func fail(err error) { fmt.Fprint(os.Stderr, err); os.Exit(1) }
