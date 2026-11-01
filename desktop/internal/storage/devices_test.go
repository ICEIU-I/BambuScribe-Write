package storage

import (
	"bambuscribe/desktop/internal/printer"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLegacyPrinterMigratesWithoutRewritingDraft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	old := Data{Version: 1, Preferences: Defaults(), Printer: printer.Config{IP: "127.0.0.1", AccessCode: "visible-code", Serial: "STALE12345"}}
	raw, _ := json.Marshal(old)
	var data map[string]json.RawMessage
	json.Unmarshal(raw, &data)
	delete(data, "devices")
	delete(data, "selectedDeviceId")
	raw, _ = json.Marshal(data)
	os.WriteFile(path, raw, 0600)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	d := s.Snapshot()
	beforePrefs, _ := json.Marshal(old.Preferences)
	afterPrefs, _ := json.Marshal(d.Preferences)
	if len(d.Devices) != 1 || d.SelectedDeviceID != "primary" || d.Devices[0].AccessCode != "visible-code" || string(beforePrefs) != string(afterPrefs) {
		t.Fatal("migration lost existing settings")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(raw) {
		t.Fatal("opening rewrote original settings")
	}
	if err := s.SaveDevices([]printer.Device{}, ""); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil || len(reopened.Snapshot().Devices) != 0 || reopened.Snapshot().Printer.IP != "" {
		t.Fatal("deleted last device was resurrected")
	}
}
func TestDeviceProfilesRoundTripAndRejectDuplicateIPs(t *testing.T) {
	s, _ := Open(t.TempDir())
	devices := []printer.Device{{ID: "one", Name: "左", Config: printer.Config{IP: "127.0.0.1", AccessCode: "one"}}, {ID: "two", Name: "右", Config: printer.Config{IP: "127.0.0.2", AccessCode: "two"}}}
	if err := s.SaveDevices(devices, "two"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(filepath.Dir(s.path))
	if err != nil || !reflect.DeepEqual(reopened.Snapshot().Devices, devices) || reopened.Snapshot().SelectedDeviceID != "two" {
		t.Fatal("profiles did not round trip")
	}
	devices[1].IP = devices[0].IP
	if err := s.SaveDevices(devices, "two"); err == nil {
		t.Fatal("same printer allowed in two profiles")
	}
	if s.Snapshot().Devices[1].IP != "127.0.0.2" {
		t.Fatal("invalid edit changed saved profiles")
	}
}
