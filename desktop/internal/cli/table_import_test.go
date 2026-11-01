package cli

import (
	"bambuscribe/desktop/internal/liveeditor"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const importExample = `{"extraction":{"tables":[{"title":"实验表","notes":[],"rows":1,"columns":[1],"style":"grid","cells":[{"row":0,"col":0,"rowSpan":1,"colSpan":1,"text":"U_{AK}/V","uncertain":false,"borders":null}]}]},"placement":{"page":1,"y":200}}`

func TestImportCommandSavesEditableObjectsAndPreservesFileOnFailure(t *testing.T) {
	p := newProject(t)
	out := filepath.Join(t.TempDir(), "imported.ccf")
	run(t, New(), []string{"import-table", "--project", p, "--output", out, "--params-file", "-"}, importExample)
	got, err := load(out)
	if err != nil || len(got.Tables) != 2 || got.Tables[1].Cells[0].Text != "U_{AK}/V" {
		t.Fatal(got, err)
	}
	before, _ := os.ReadFile(out)
	bad := strings.Replace(importExample, `"colSpan":1`, `"colSpan":2`, 1)
	var buf bytes.Buffer
	if err = New().Run(context.Background(), []string{"import-table", "--project", p, "--output", out, "--params-file", "-"}, strings.NewReader(bad), &buf, &buf); err == nil {
		t.Fatal("bad import accepted")
	}
	after, _ := os.ReadFile(out)
	if !bytes.Equal(before, after) {
		t.Fatal("file changed on failure")
	}
}
func TestLiveCLIUsesExplicitSession(t *testing.T) {
	dir := t.TempDir()
	var s *liveeditor.Server
	var err error
	s, err = liveeditor.Start(dir, func(req liveeditor.Request) {
		s.Complete(req.ID, liveeditor.Reply{Result: json.RawMessage(`{"revision":3}`)})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.SetReady(true)
	output := run(t, New(), []string{"live", "snapshot", "--session", filepath.Join(dir, "editor-session.json")}, "")
	if !strings.Contains(output, `"revision":3`) {
		t.Fatal(output)
	}
}
