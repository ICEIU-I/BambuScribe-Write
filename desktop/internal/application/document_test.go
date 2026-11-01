package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/printer"
	"bambuscribe/desktop/internal/storage"
	"encoding/json"
	"strings"
	"testing"
)

func TestPreviewCacheAndExplicitDeviceCredentials(t *testing.T) {
	// No shell: the test never touches the menu bar.
	a, _, _ := New(nil)
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.store = s
	if err = s.SavePrinter(printer.Config{IP: "127.0.0.1", AccessCode: "secret-fixture", Serial: "TEST123456"}); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(a.Bootstrap())
	if !strings.Contains(string(b), "secret-fixture") || len(a.Bootstrap().Devices) != 1 {
		t.Fatal("saved instance access code must be available for direct editing")
	}
	d, err := a.Preview("你好", engine.DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	if found, err := a.document(d.ID); err != nil || found.ID != d.ID {
		t.Fatal("preview version not cached")
	}
	for _, text := range []string{"第一份", "第二份", "第三份"} {
		if _, err = a.Preview(text, engine.DefaultLayout()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = a.document(d.ID); err == nil {
		t.Fatal("preview cache unbounded")
	}
}
