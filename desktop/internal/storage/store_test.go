package storage

import (
	"bambuscribe/desktop/internal/printer"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripAndPrivateConfig(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := s.Snapshot().Preferences
	p.Theme = "dark"
	p.Text = "草稿\n第二行"
	p.Layout.FontSize = 8
	if err = s.SavePreferences(p); err != nil {
		t.Fatal(err)
	}
	if err = s.SavePrinter(printer.Config{IP: "127.0.0.1", AccessCode: "test", Serial: "TEST123456"}); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Snapshot().Preferences.Text != p.Text || loaded.Snapshot().Printer.AccessCode != "test" || loaded.Snapshot().Preferences.Theme != "dark" {
		t.Fatal("state not restored")
	}
	info, _ := os.Stat(filepath.Join(dir, "settings.json"))
	if info.Mode().Perm() != 0600 {
		t.Fatal("credentials file permissions")
	}
}

func TestCorruptConfigIsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte("broken"), 0600)
	s, err := Open(dir)
	if err == nil {
		t.Fatal("missing read error")
	}
	if err = s.SavePreferences(Defaults()); err == nil {
		t.Fatal("corrupt original overwritten")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "broken" {
		t.Fatal("original not preserved")
	}
}
