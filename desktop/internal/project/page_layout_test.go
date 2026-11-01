package project

import (
	"bambuscribe/desktop/internal/engine"
	"reflect"
	"testing"
)

func TestProjectRetainsExplicitPages(t *testing.T) {
	p := testProject(t)
	p.Text = "第一页\f复制页"
	p.Layout.PageLayouts = []engine.PageLayout{{LineCount: 16}, {LineCount: 16, Continuation: true}}
	raw, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Text != p.Text || !reflect.DeepEqual(restored.Layout.PageLayouts, p.Layout.PageLayouts) {
		t.Fatal("page boundary or grid lost across save")
	}
}
