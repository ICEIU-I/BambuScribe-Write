package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"bambuscribe/desktop/internal/storage"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func persistenceApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &App{store: s}, dir
}

func saveFixtureAt(t *testing.T, a *App, p storage.Preferences, path string) error {
	t.Helper()
	document := projectFromPreferences(p)
	data, err := project.Encode(document)
	if err != nil {
		t.Fatal(err)
	}
	a.persistenceMu.Lock()
	defer a.persistenceMu.Unlock()
	_, err = a.saveProjectAt(document, data, path)
	return err
}

func readFixture(t *testing.T, path string) project.Project {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	p, err := project.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWorkspaceAutosavesOnlyLatestSuccessfulProject(t *testing.T) {
	a, storeDir := persistenceApp(t)
	p := storage.Defaults()
	p.Text, p.Theme = "首次草稿", "dark"
	p.Layout.PageLayouts = []engine.PageLayout{{LineCount: 12}}
	p.Erasures = []engine.PageErasure{{ID: "e", Page: 0, X: 1, Y: 1, Width: 2, Height: 2}}
	if err := a.SavePreferences(p, ""); err != nil {
		t.Fatal(err)
	}
	if a.store.Snapshot().Preferences.Text != p.Text {
		t.Fatal("an unnamed document must still retain its draft")
	}
	dir := t.TempDir()
	first, latest := filepath.Join(dir, "first.ccf"), filepath.Join(dir, "latest.ccf")
	if err := saveFixtureAt(t, a, p, first); err != nil {
		t.Fatal(err)
	}
	p.Text = "首次保存后编辑"
	if err := a.SavePreferences(p, first); err != nil {
		t.Fatal(err)
	}
	if readFixture(t, first).Text != p.Text {
		t.Fatal("autosave did not update the named file")
	}
	if err := saveFixtureAt(t, a, p, latest); err != nil {
		t.Fatal(err)
	}
	firstBytes, _ := os.ReadFile(first)
	if err := saveFixtureAt(t, a, p, filepath.Join(dir, "missing", "failed.ccf")); err == nil {
		t.Fatal("expected a failed save-as")
	}
	p.Text = "最近一次保存后的编辑"
	if err := a.SavePreferences(p, latest); err != nil {
		t.Fatal(err)
	}
	if readFixture(t, latest).Text != p.Text || a.projectPath() != latest {
		t.Fatal("a failed save-as replaced the last successful destination")
	}
	unchanged, _ := os.ReadFile(first)
	if !reflect.DeepEqual(firstBytes, unchanged) {
		t.Fatal("autosave rewrote an older copy")
	}
	d := a.store.Snapshot()
	if d.ProjectPath != latest || d.Preferences.Text != "" || len(d.Preferences.Images)+len(d.Preferences.Tables)+len(d.Preferences.Erasures)+len(d.Preferences.Layout.PageLayouts) != 0 {
		t.Fatal("named project still has a parallel document draft")
	}
	loaded, err := storage.Open(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	restarted := &App{store: loaded}
	restarted.setProjectPath(loaded.Snapshot().ProjectPath)
	restored, path, warning := restarted.workspaceSnapshot()
	if warning != "" || path != latest || !reflect.DeepEqual(restored.Preferences, p) {
		t.Fatalf("restart did not load the actual project: %s, %s", path, warning)
	}
}

func TestWorkspaceRejectsDelayedSaveAndRetainsFailedFile(t *testing.T) {
	a, _ := persistenceApp(t)
	p := storage.Defaults()
	path := filepath.Join(t.TempDir(), "current.ccf")
	if err := saveFixtureAt(t, a, p, path); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	p.Text = "旧项目的延迟保存"
	if err := a.SavePreferences(p, "/tmp/older.ccf"); err == nil {
		t.Fatal("accepted a stale destination")
	}
	after, _ := os.ReadFile(path)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("stale autosave touched the current file")
	}
	if err := os.Rename(filepath.Dir(path), filepath.Dir(path)+"-moved"); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path) + "-moved")
	if err := a.SavePreferences(p, path); err == nil {
		t.Fatal("missing target directory must report a save failure")
	}
	if a.projectPath() != path || a.store.Snapshot().Preferences.Text != "" {
		t.Fatal("failed autosave fell back to a draft")
	}
}

func TestWorkspaceMissingOrCorruptProjectNeverReceivesFallbackEdits(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "corrupt"}[corrupt], func(t *testing.T) {
			a, _ := persistenceApp(t)
			path := filepath.Join(t.TempDir(), "unreadable.ccf")
			if corrupt {
				if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a.setProjectPath(path)
			_, restoredPath, warning := a.workspaceSnapshot()
			if restoredPath != "" || warning == "" {
				t.Fatal("failed project read left the fallback bound to the file")
			}
			if err := a.SavePreferences(storage.Defaults(), path); err == nil {
				t.Fatal("fallback autosave replaced an unreadable file")
			}
			if corrupt {
				data, _ := os.ReadFile(path)
				if string(data) != "broken" {
					t.Fatal("damaged file was overwritten")
				}
			} else if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("missing project was silently recreated")
			}
		})
	}
}

func TestNewWorkspaceClearsProjectReferenceWithoutChangingOldFile(t *testing.T) {
	a, _ := persistenceApp(t)
	path := filepath.Join(t.TempDir(), "previous.ccf")
	if err := saveFixtureAt(t, a, storage.Defaults(), path); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := a.NewProject(); err != nil {
		t.Fatal(err)
	}
	if a.store.Snapshot().ProjectPath != "" || a.projectPath() != "" {
		t.Fatal("new project still targets the previous file")
	}
	p := storage.Defaults()
	p.Text = "独立的新草稿"
	if err := a.SavePreferences(p, ""); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !reflect.DeepEqual(before, after) || a.store.Snapshot().Preferences.Text != p.Text {
		t.Fatal("new draft changed the old file")
	}
}
