package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/printer"
	"bambuscribe/desktop/internal/storage"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

type instanceTestLink struct {
	mu     sync.Mutex
	closed int
	files  []string
}

func (l *instanceTestLink) Connect(_ context.Context, _ printer.Config, report func(map[string]any), _ func(error)) error {
	report(map[string]any{"gcode_state": "IDLE"})
	return nil
}
func (l *instanceTestLink) Publish(map[string]any) error { return nil }
func (l *instanceTestLink) Upload(_ context.Context, name string, _ []byte, progress func(int)) error {
	l.mu.Lock()
	l.files = append(l.files, name)
	l.mu.Unlock()
	progress(100)
	return nil
}
func (l *instanceTestLink) Close() { l.mu.Lock(); l.closed++; l.mu.Unlock() }
func TestInstanceCRUDRoutingAndSettingsIsolation(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	prefs := store.Snapshot().Preferences
	app := &App{ctx: context.Background(), store: store, documents: map[string]*engine.Document{}}
	links := []*instanceTestLink{}
	app.fleet = printer.NewFleet(func() printer.Link { l := &instanceTestLink{}; links = append(links, l); return l }, func(context.Context, string) (string, error) { return "DETECTED123", nil }, nil)
	defer app.fleet.Close()
	first, err := app.SavePrinterDevice(printer.Device{Name: "左 A1", Config: printer.Config{IP: "127.0.0.1", AccessCode: "first-code"}})
	if err != nil {
		t.Fatal(err)
	}
	id1 := first.Devices[0].ID
	second, err := app.SavePrinterDevice(printer.Device{Name: "右 A1", Config: printer.Config{IP: "127.0.0.2", AccessCode: "second-code"}})
	if err != nil {
		t.Fatal(err)
	}
	id2 := second.Devices[1].ID
	if err := app.ConnectDevice(id1); err != nil {
		t.Fatal(err)
	}
	if err := app.ConnectDevice(id2); err != nil {
		t.Fatal(err)
	}
	rename := second.Devices[0].Device
	rename.Name = "实验台"
	if _, err := app.SavePrinterDevice(rename); err != nil {
		t.Fatal(err)
	}
	if !app.fleet.State(id1).Connected || !app.fleet.State(id2).Connected {
		t.Fatal("renaming closed a connection")
	}
	path := filepath.Join(dir, "settings.json")
	before, _ := os.ReadFile(path)
	if _, err := app.SavePrinterDevice(rename); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("unchanged metadata rewrote configuration")
	}
	document := uploadDocument()
	app.documents[document.ID] = document
	if _, err := app.UploadDevicePages(id2, document.ID, "31", nil, engine.DefaultWriting()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for app.fleet.State(id2).Phase == "uploading" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if app.fleet.State(id2).Phase != "uploaded" {
		t.Fatal("upload did not finish")
	}
	links[0].mu.Lock()
	wrong := len(links[0].files)
	links[0].mu.Unlock()
	links[1].mu.Lock()
	count := len(links[1].files)
	links[1].mu.Unlock()
	if wrong != 0 || count != 2 {
		t.Fatal("upload routed to wrong instance")
	}
	rename.IP = "127.0.0.3"
	if _, err := app.SavePrinterDevice(rename); err != nil {
		t.Fatal(err)
	}
	if app.fleet.State(id1).Connected || !app.fleet.State(id2).Connected {
		t.Fatal("editing IP affected another device")
	}
	if _, err := app.SelectPrinter(id2); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DeletePrinterDevice(id1); err != nil {
		t.Fatal(err)
	}
	if !app.fleet.State(id2).Connected || len(app.Printers().Devices) != 1 {
		t.Fatal("delete affected another instance")
	}
	if _, err := app.DeletePrinterDevice(id2); err != nil {
		t.Fatal(err)
	}
	if len(app.Printers().Devices) != 0 || app.Printers().SelectedDeviceID != "" {
		t.Fatal("last deletion not reflected")
	}
	if !reflect.DeepEqual(store.Snapshot().Preferences, prefs) {
		t.Fatal("device operations changed the document or calibration")
	}
}
