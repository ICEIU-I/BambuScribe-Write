package printer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeLink struct {
	mu       sync.Mutex
	payloads []map[string]any
}

func (f *fakeLink) Connect(_ context.Context, _ Config, _ func(map[string]any), _ func(error)) error {
	return nil
}
func (f *fakeLink) Publish(payload map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payloads = append(f.payloads, payload)
	return nil
}
func (f *fakeLink) Upload(_ context.Context, _ string, _ []byte, progress func(int)) error {
	progress(100)
	return nil
}
func (f *fakeLink) Close() {}

func setupManager(t *testing.T) (*Manager, *fakeLink) {
	t.Helper()
	f := &fakeLink{}
	m := NewManager(f, nil)
	if err := m.Connect(context.Background(), Config{IP: "127.0.0.1", Serial: "TEST123456", AccessCode: "test"}); err != nil {
		t.Fatal(err)
	}
	m.Observe(map[string]any{"gcode_state": "IDLE"})
	return m, f
}
func awaitPhase(t *testing.T, m *Manager, phase string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if m.State().Phase == phase {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("want %s, got %+v", phase, m.State())
}
func assertNoWritingCommands(t *testing.T, f *fakeLink) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, payload := range f.payloads {
		if _, ok := payload["print"]; ok {
			t.Fatalf("client issued a writing command: %+v", payload)
		}
	}
}

func TestConnectionAndDeviceReportsNeverControlWriting(t *testing.T) {
	m, f := setupManager(t)
	for _, state := range []string{"RUNNING", "PAUSE"} {
		m.Observe(map[string]any{"gcode_state": state})
		if m.State().Phase != "external" {
			t.Fatal(m.State())
		}
		if _, err := m.UploadPages("doc", "31", threePages); err == nil {
			t.Fatal("busy device accepted upload")
		}
	}
	m.Observe(map[string]any{"gcode_state": "FINISH"})
	if m.State().Phase != "idle" {
		t.Fatal(m.State())
	}
	assertNoWritingCommands(t, f)
	m.lost(errors.New("lost"))
	if m.State().Connected || m.State().Phase != "disconnected" {
		t.Fatal("disconnect not reflected")
	}
	m.Observe(map[string]any{"gcode_state": "IDLE"})
	if _, err := m.UploadPages("doc", "31", threePages); err == nil {
		t.Fatal("disconnected upload accepted")
	}
}

func TestUploadWaitsForDeviceStatusAndCancellationRequiresTransfer(t *testing.T) {
	m, _ := setupManager(t)
	m.mu.Lock()
	m.state.DeviceState = ""
	m.mu.Unlock()
	if _, err := m.UploadPages("doc", "31", threePages); err == nil {
		t.Fatal("upload accepted without device status")
	}
	if err := m.CancelUpload(); err == nil {
		t.Fatal("cancel accepted without transfer")
	}
}

func TestManualDisconnectDoesNotReportConnectionFailure(t *testing.T) {
	m, _ := setupManager(t)
	m.Close()
	if m.State().Connected || m.State().Phase != "disconnected" || m.State().Message != "" {
		t.Fatal("manual disconnect was presented as a connection failure")
	}
}

func TestDiscoveryAndInputValidation(t *testing.T) {
	if serialFromCN("ABC123456.bambulab.com") != "ABC123456" {
		t.Fatal("certificate parsing")
	}
	if serialFromCN("abc/123") != "" {
		t.Fatal("unsafe serial")
	}
	if err := (Config{IP: "127.0.0.1", AccessCode: "x\nPASS", Serial: "ABC123456"}).Validate(); err == nil {
		t.Fatal("control characters allowed")
	}
	r := &progressReader{ctx: context.Background(), reader: strings.NewReader("123"), total: 3, progress: func(p int) {
		if p > 99 {
			t.Fatal("reader claimed confirmed upload")
		}
	}}
	r.Read(make([]byte, 4))
}
