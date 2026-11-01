package cli

import (
	"bambuscribe/desktop/internal/engine"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, r *Runner, args []string, input string) string {
	t.Helper()
	var out, progress bytes.Buffer
	if err := r.Run(context.Background(), args, strings.NewReader(input), &out, &progress); err != nil {
		t.Fatalf("%v: %v (%s)", args, err, progress.String())
	}
	return out.String()
}
func newProject(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "note.ccf")
	run(t, New(), []string{"new", "--output", path}, "")
	return path
}

func TestFileWorkflowAndPreviewExportMatchDesktopVectors(t *testing.T) {
	source := newProject(t)
	target := filepath.Join(t.TempDir(), "edited.ccf")
	ops := `[
	{"tool":"table.insert","params":{"id":"table","rows":2,"cols":2,"x":15,"y":45,"width":100,"fontSize":4}},
	{"tool":"table.paste","params":{"id":"table","data":[["实验","数据"],["1","2"]]}},
	{"tool":"textbox.insert","params":{"id":"title","text":"表1 实验数据","x":15,"y":20,"width":130}},
	{"tool":"layout.set","params":{"letterSpacing":85}},
	{"tool":"writing.set","params":{"sound":false}}
	]`
	run(t, New(), []string{"batch", "--project", source, "--output", target, "--ops-file", "-"}, ops)
	p, err := load(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Tables) != 2 || p.Tables[1].Kind != "textbox" || p.Tables[0].Cells[3].Text != "2" || p.Layout.LetterSpacing != 85 || p.Writing.Sound {
		t.Fatal("saved edits lost")
	}
	if p.Tables[1].FontSize != p.Layout.FontSize {
		t.Fatal("CLI text box did not inherit the document font size")
	}
	if original, _ := load(source); len(original.Tables) != 0 {
		t.Fatal("source file altered")
	}
	inspection := run(t, New(), []string{"inspect", "--project", target}, "")
	if !strings.Contains(inspection, `"kind":"textbox"`) {
		t.Fatal(inspection)
	}
	run(t, New(), []string{"check", "--project", target}, "")
	file := filepath.Join(t.TempDir(), "page.gcode")
	run(t, New(), []string{"export", "--project", target, "--page", "1", "--output", file}, "")
	d, _ := render(p)
	expected, err := engine.ExportPage(d, 0, p.Writing)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(file)
	if !bytes.Equal(expected, actual) {
		t.Fatal("CLI export differs from engine")
	}
	svg := run(t, New(), []string{"preview", "--project", target, "--format", "svg", "--output", "-"}, "")
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		Groups  []struct {
			Paths []struct {
				D string `xml:"d,attr"`
			} `xml:"path"`
		} `xml:"g"`
	}
	if err = xml.Unmarshal([]byte(svg), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Groups[0].Paths) != len(d.Pages[0].Strokes) {
		t.Fatal("SVG omits strokes")
	}
	zip := filepath.Join(t.TempDir(), "all.zip")
	run(t, New(), []string{"export", "--project", target, "--page", "all", "--output", zip}, "")
	run(t, New(), []string{"save", "--project", target, "--output", source}, "")
	if restored, _ := load(source); len(restored.Tables) != 2 {
		t.Fatal("save as lost objects")
	}
}

func TestFailedBatchAndExportNeverReplaceOutput(t *testing.T) {
	p := newProject(t)
	before, _ := os.ReadFile(p)
	var out, errout bytes.Buffer
	err := New().Run(context.Background(), []string{"batch", "--project", p, "--output", p, "--ops-file", "-"}, strings.NewReader(`[{"tool":"textbox.insert","params":{"id":"a"}},{"tool":"object.move","params":{"id":"missing"}}]`), &out, &errout)
	if err == nil {
		t.Fatal("bad batch succeeded")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("partial batch replaced project")
	}
	run(t, New(), []string{"tool", "textbox.insert", "--project", p, "--output", p, "--params-file", "-"}, `{"id":"title","text":"🙂"}`)
	file := filepath.Join(t.TempDir(), "page.gcode")
	_ = os.WriteFile(file, []byte("keep"), 0600)
	err = New().Run(context.Background(), []string{"export", "--project", p, "--output", file}, strings.NewReader(""), &out, &errout)
	if err == nil {
		t.Fatal("missing glyph exported")
	}
	b, _ := os.ReadFile(file)
	if string(b) != "keep" {
		t.Fatal("failed export replaced file")
	}
	err = New().Run(context.Background(), []string{"check", "--project", p}, strings.NewReader(""), &out, &errout)
	if err == nil {
		t.Fatal("failed preflight returned success")
	}
}

func TestHelpCatalogStrictParamsAndCancellation(t *testing.T) {
	r := New()
	help := run(t, r, []string{"--help"}, "")
	if !strings.Contains(help, "device upload") {
		t.Fatal(help)
	}
	var catalog []ToolSpec
	if err := json.Unmarshal([]byte(run(t, r, []string{"tools"}, "")), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog) < 30 {
		t.Fatal("incomplete tools")
	}
	p := newProject(t)
	var out bytes.Buffer
	for _, args := range [][]string{
		{"tool", "textbox.insert", "--project", p, "--output", p, "--params", `{"fontzise":4}`},
		{"tool", "text.set", "--project", p, "--output", p, "--params", `{"text":"x"} {}`},
		{"tool", "text.set", "--project", p, "--output", p, "--params", `{}`, "--params-file", "-"},
		{"tool", "text.set", "--project", p, "--output", p, "--params", `{"text":"x"}`, "--typo"},
	} {
		if err := r.Run(context.Background(), args, strings.NewReader("{}"), &out, &out); err == nil {
			t.Fatalf("accepted bad call %v", args)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.Run(ctx, []string{"inspect", "--project", p}, strings.NewReader(""), &out, &out); err != context.Canceled {
		t.Fatal(err)
	}
}
