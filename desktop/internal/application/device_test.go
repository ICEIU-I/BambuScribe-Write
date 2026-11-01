package application

import (
	"bambuscribe/desktop/internal/printer"
	"bambuscribe/desktop/internal/storage"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type savedDeviceLink struct {
	configs  []printer.Config
	requests []map[string]any
	err      error
}

func (l *savedDeviceLink) Connect(_ context.Context, c printer.Config, _ func(map[string]any), _ func(error)) error {
	l.configs = append(l.configs, c)
	return l.err
}
func (l *savedDeviceLink) Publish(p map[string]any) error {
	l.requests = append(l.requests, p)
	return nil
}
func (l *savedDeviceLink) Upload(context.Context, string, []byte, func(int)) error {
	return errors.New("unexpected upload during connection")
}
func (l *savedDeviceLink) Close() {}

func TestSavedPrinterReconnectReusesSecretAndPreservesConfigBytes(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "connected", true: "offline"}[fail], func(t *testing.T) {
			dir := t.TempDir()
			store, err := storage.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			prefs := storage.Defaults()
			prefs.Text = "正文、校准与配置保持原样"
			up := 9.0
			prefs.Writing.PenUp = &up
			if err = store.SavePreferences(prefs); err != nil {
				t.Fatal(err)
			}
			config := printer.Config{IP: "192.168.1.100", Serial: "TESTA1SERIAL", AccessCode: "test-secret"}
			if err = store.SavePrinter(config); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "settings.json")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			// Keep valid JSON with a distinct representation: an unnecessary save
			// would remove this trailing newline even if all values stayed equal.
			before = append(before, '\n')
			if err = os.WriteFile(path, before, 0600); err != nil {
				t.Fatal(err)
			}
			link := &savedDeviceLink{}
			if fail {
				link.err = errors.New("connection timed out")
			}
			app := &App{ctx: context.Background(), store: store}
			detections := 0
			app.fleet = printer.NewFleet(func() printer.Link { return link }, func(_ context.Context, ip string) (string, error) {
				detections++
				if ip != config.IP {
					t.Fatalf("detecting a different IP: %s", ip)
				}
				return config.Serial, nil
			}, nil)
			app.fleet.Add("primary")
			err = app.ConnectDevice("primary")
			if (err != nil) != fail {
				t.Fatalf("unexpected connection result: %v", err)
			}
			if detections != 1 || len(link.configs) != 1 || link.configs[0] != config {
				t.Fatal("saved device credentials were not reused")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(store.Snapshot().Preferences, prefs) {
				t.Fatal("reconnect changed draft or config")
			}
			if !fail {
				want := []map[string]any{{"pushing": map[string]any{"sequence_id": "0", "command": "pushall"}}}
				if !reflect.DeepEqual(link.requests, want) {
					t.Fatal("connecting should only request device status")
				}
			} else if len(link.requests) != 0 {
				t.Fatal("requests sent after a failed connection")
			}
		})
	}
}

func TestPrinterReconnectDetectsCurrentSerialEveryTime(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	old := printer.Config{IP: "192.168.1.100", Serial: "OLDA1SERIAL", AccessCode: "test-secret"}
	if err = store.SavePrinter(old); err != nil {
		t.Fatal(err)
	}
	prefs := store.Snapshot().Preferences
	link := &savedDeviceLink{}
	app := &App{ctx: context.Background(), store: store}
	serials := []string{"FIRSTA1SERIAL", "SECONDA1SERIAL"}
	detections := 0
	discover := func(_ context.Context, ip string) (string, error) {
		if ip != old.IP {
			t.Fatalf("detecting a different IP: %s", ip)
		}
		sn := serials[detections]
		detections++
		return sn, nil
	}
	app.fleet = printer.NewFleet(func() printer.Link { return link }, discover, nil)
	app.fleet.Add("primary")
	for _, serial := range serials {
		// Even an explicitly supplied stale serial must not skip detection.
		if err = app.ConnectDevice("primary"); err != nil {
			t.Fatal(err)
		}
		want := printer.Config{IP: old.IP, Serial: serial, AccessCode: old.AccessCode}
		if link.configs[len(link.configs)-1] != want {
			t.Fatal("connection did not use the newly detected serial")
		}
		app.DisconnectPrinter()
	}
	if detections != 2 || len(link.configs) != 2 || !reflect.DeepEqual(store.Snapshot().Preferences, prefs) {
		t.Fatal("reconnect skipped detection or changed the draft")
	}
}

func TestPrinterDetectionFailureDoesNotUseSavedSerial(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	config := printer.Config{IP: "192.168.1.100", Serial: "SAVEDA1SERIAL", AccessCode: "test-secret"}
	if err = store.SavePrinter(config); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "settings.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	link := &savedDeviceLink{}
	app := &App{ctx: context.Background(), store: store}
	detectionError := errors.New("cannot detect current printer")
	app.fleet = printer.NewFleet(func() printer.Link { return link }, func(context.Context, string) (string, error) { return "", detectionError }, nil)
	app.fleet.Add("primary")
	err = app.ConnectDevice("primary")
	if !errors.Is(err, detectionError) || len(link.configs) != 0 || len(link.requests) != 0 {
		t.Fatal("failed detection used a stale serial or sent device requests")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed detection changed the saved config")
	}
}
