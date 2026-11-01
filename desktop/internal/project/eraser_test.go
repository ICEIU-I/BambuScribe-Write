package project

import (
	"bambuscribe/desktop/internal/engine"
	"reflect"
	"testing"
)

func TestProjectRetainsPaperErasureGeometry(t *testing.T) {
	p := testProject(t)
	p.Erasures = []engine.PageErasure{{ID: "rectangle", Page: 1, X: 12, Y: 25, Width: 30, Height: 20}, {ID: "path", X: 1, Y: 2, Width: 2, Height: 2, Path: []engine.Point{{X: 1, Y: 2}, {X: 3, Y: 4}}, Radius: 1.8}}
	targets, bodyOnly := []string{"original-image", "original-table"}, []string{}
	p.Erasures[0].ObjectIDs, p.Erasures[1].ObjectIDs = &targets, &bodyOnly
	cuts := []engine.InkCut{{ObjectID: "original-image", Part: "graphic", Source: "source", Stroke: 3, Ranges: [][2]float64{{.25, .75}, {1.2, 2}}}}
	empty := []engine.InkCut{}
	p.Erasures[0].Cuts, p.Erasures[1].Cuts = &cuts, &empty
	raw, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(raw)
	if err != nil || !reflect.DeepEqual(p.Erasures, restored.Erasures) {
		t.Fatal("project lost erasure geometry", err)
	}
	p.Erasures[1].Radius = -1
	if _, err := Encode(p); err == nil {
		t.Fatal("saved invalid brush")
	}
}
