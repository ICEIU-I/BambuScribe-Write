package cli

import (
	"bambuscribe/desktop/internal/printer"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type fakeLink struct {
	mu         sync.Mutex
	report     func(map[string]any)
	lost       func(error)
	files      []string
	commands   []map[string]any
	code       string
	fail       int
	block      bool
	entered    chan struct{}
	closed     bool
	connectErr error
}

func (f *fakeLink) Connect(_ context.Context, c printer.Config, report func(map[string]any), lost func(error)) error {
	f.report, f.lost, f.code = report, lost, c.AccessCode
	return f.connectErr
}
func (f *fakeLink) Publish(p map[string]any) error {
	f.mu.Lock()
	f.commands = append(f.commands, p)
	f.mu.Unlock()
	f.report(map[string]any{"gcode_state": "IDLE"})
	return nil
}
func (f *fakeLink) Upload(ctx context.Context, name string, _ []byte, progress func(int)) error {
	f.mu.Lock()
	f.files = append(f.files, name)
	n := len(f.files)
	f.mu.Unlock()
	progress(50)
	if f.entered != nil {
		f.entered <- struct{}{}
	}
	if f.block {
		<-ctx.Done()
		return ctx.Err()
	}
	if f.fail == n {
		return fmt.Errorf("failed with %s", f.code)
	}
	progress(100)
	return nil
}
func (f *fakeLink) Close() { f.mu.Lock(); f.closed = true; f.mu.Unlock() }

func deviceRunner(t *testing.T, f *fakeLink) (*Runner, string) {
	t.Helper()
	config := filepath.Join(t.TempDir(), "device.json")
	_ = os.WriteFile(config, []byte(`{"ip":"127.0.0.1","serial":"TEST123456"}`), 0600)
	r := New()
	r.NewLink = func() printer.Link { return f }
	r.Getenv = func(string) string { return "secret-value" }
	return r, config
}
func uploadProject(t *testing.T) string {
	p := newProject(t)
	run(t, New(), []string{"batch", "--project", p, "--output", p, "--ops-file", "-"}, `[{"tool":"textbox.insert","params":{"id":"a","text":"实验","page":1}},{"tool":"textbox.insert","params":{"id":"b","text":"数据","page":3}}]`)
	return p
}
func assertOnlyStatusCommands(t *testing.T, f *fakeLink) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.commands {
		if len(p) != 1 || p["pushing"] == nil {
			t.Fatalf("unexpected device command %v", p)
		}
	}
	if !f.closed {
		t.Fatal("connection left open")
	}
}

func TestDeviceStatusAndWholeBatchUploadPreservePageNumbers(t *testing.T) {
	for _, name := range []string{"connect", "status", "upload"} {
		t.Run(name, func(t *testing.T) {
			f := &fakeLink{}
			r, c := deviceRunner(t, f)
			args := []string{"device", name, "--config", c, "--timeout", "2s"}
			if name == "upload" {
				args = append(args, "--project", uploadProject(t), "--params", `{"prefix":"31"}`)
			}
			result := run(t, r, args, "")
			if strings.Contains(result, "secret-value") {
				t.Fatal("credential leak")
			}
			if name == "upload" {
				f.mu.Lock()
				files := strings.Join(f.files, ",")
				f.mu.Unlock()
				if files != "31-page1.gcode,31-page3.gcode" {
					t.Fatal(files)
				}
			}
			assertOnlyStatusCommands(t, f)
		})
	}
}

func TestUploadFailureStopsBatchAndRedactsEveryStream(t *testing.T) {
	f := &fakeLink{fail: 1}
	r, c := deviceRunner(t, f)
	var out, progress bytes.Buffer
	err := r.Run(context.Background(), []string{"device", "upload", "--config", c, "--project", uploadProject(t), "--params", `{"prefix":"31"}`}, strings.NewReader(""), &out, &progress)
	if err == nil {
		t.Fatal("failed transfer succeeded")
	}
	if strings.Contains(err.Error()+out.String()+progress.String(), "secret-value") {
		t.Fatal("credential leak")
	}
	f.mu.Lock()
	n := len(f.files)
	f.mu.Unlock()
	if n != 1 {
		t.Fatal("continued after failure")
	}
	assertOnlyStatusCommands(t, f)
}

func TestUploadPreflightHappensBeforeAnyDeviceConnection(t *testing.T) {
	p := uploadProject(t)
	run(t, New(), []string{"tool", "textbox.update", "--project", p, "--output", p, "--params", `{"id":"b","text":"🙂"}`}, "")
	f := &fakeLink{}
	r, c := deviceRunner(t, f)
	connections := 0
	r.NewLink = func() printer.Link { connections++; return f }
	var out bytes.Buffer
	err := r.Run(context.Background(), []string{"device", "upload", "--config", c, "--project", p, "--params", `{"prefix":"31"}`}, strings.NewReader(""), &out, &out)
	if err == nil || connections != 0 {
		t.Fatal("bad later page connected/transferred", err, connections)
	}
	run(t, r, []string{"device", "upload", "--config", c, "--project", p, "--params", `{"prefix":"31","pages":[1]}`}, "")
	if connections != 1 {
		t.Fatal(connections)
	}
}

func TestCancellationAndDisconnectReleaseUpload(t *testing.T) {
	for _, action := range []string{"cancel", "disconnect"} {
		t.Run(action, func(t *testing.T) {
			f := &fakeLink{block: true, entered: make(chan struct{}, 1)}
			r, c := deviceRunner(t, f)
			p := uploadProject(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			var out, progress bytes.Buffer
			go func() {
				done <- r.Run(ctx, []string{"device", "upload", "--config", c, "--project", p, "--params", `{"prefix":"31"}`}, strings.NewReader(""), &out, &progress)
			}()
			<-f.entered
			if action == "cancel" {
				cancel()
			} else {
				f.lost(errors.New("disconnected"))
			}
			err := <-done
			if err == nil {
				t.Fatal("interruption returned success")
			}
			if action == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			assertOnlyStatusCommands(t, f)
		})
	}
}

func TestDiscoveryAndWatchEmitMachineReadableState(t *testing.T) {
	f := &fakeLink{}
	r, c := deviceRunner(t, f)
	r.Discover = func(_ context.Context, ip string) (string, error) { return "TEST123456", nil }
	result := run(t, r, []string{"device", "discover", "--ip", "127.0.0.1"}, "")
	if !json.Valid([]byte(result)) {
		t.Fatal(result)
	}
	var out, progress bytes.Buffer
	err := r.Run(context.Background(), []string{"device", "watch", "--config", c, "--timeout", "20ms"}, strings.NewReader(""), &out, &progress)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(out.String(), `"event":"device.state"`) {
		t.Fatal(err, out.String())
	}
	assertOnlyStatusCommands(t, f)
}
