package application

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewProjectDetachesPreviousFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "original.ccf")
	if err := os.WriteFile(path, []byte("existing project"), 0600); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	a.setProjectPath(path)
	a.NewProject()
	if a.projectPath() != "" {
		t.Fatal("new document would save over the old project")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "existing project" {
		t.Fatalf("new project touched previous file: %q, %v", data, err)
	}
}
