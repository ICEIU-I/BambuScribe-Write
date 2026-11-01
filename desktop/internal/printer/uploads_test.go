package printer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type uploadedFile struct {
	name string
	data []byte
}
type batchLink struct {
	fakeLink
	files           []uploadedFile
	failAt, blockAt int
	entered         chan int
	unblock         chan struct{}
	finished        chan int
}

func (f *batchLink) Upload(ctx context.Context, name string, data []byte, progress func(int)) error {
	f.mu.Lock()
	f.files = append(f.files, uploadedFile{name, append([]byte(nil), data...)})
	n := len(f.files)
	f.mu.Unlock()
	if f.finished != nil {
		defer func() { f.finished <- n }()
	}
	progress(50)
	if f.entered != nil {
		f.entered <- n
	}
	if n == f.blockAt {
		select {
		case <-f.unblock:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if n == f.failAt {
		return errors.New("upload interrupted")
	}
	progress(100)
	return nil
}
func batchManager(t *testing.T, f *batchLink) *Manager {
	t.Helper()
	m := NewManager(f, nil)
	if err := m.Connect(context.Background(), Config{IP: "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	m.Observe(map[string]any{"gcode_state": "IDLE"})
	return m
}

var threePages = []PageUpload{{1, []byte("first")}, {3, []byte("third")}, {4, []byte("fourth")}}

func TestBatchUploadsAllFilesWithoutStartingWriting(t *testing.T) {
	f := &batchLink{}
	m := batchManager(t, f)
	if _, err := m.UploadPages("doc", "31", threePages); err != nil {
		t.Fatal(err)
	}
	awaitPhase(t, m, "uploaded")
	f.mu.Lock()
	if len(f.files) != 3 || string(f.files[1].data) != "third" || f.files[0].name != "31-page1.gcode" || f.files[1].name != "31-page3.gcode" || f.files[2].name != "31-page4.gcode" {
		t.Fatalf("incorrect files: %+v", f.files)
	}
	f.mu.Unlock()
	s := m.State()
	if s.Page != 4 || s.PageOrdinal != 3 || s.TotalPages != 3 || s.UploadedPages != 3 || s.Progress != 100 || s.JobID != "" {
		t.Fatal(s)
	}
	// Finishing unrelated hardware tasks must not erase the upload result.
	m.Observe(map[string]any{"gcode_state": "FINISH", "subtask_name": "old-job"})
	if m.State().Phase != "uploaded" {
		t.Fatal(m.State())
	}
	m.Observe(map[string]any{"gcode_state": "RUNNING", "subtask_name": "selected-on-a1"})
	m.Observe(map[string]any{"gcode_state": "PAUSE"})
	m.Observe(map[string]any{"gcode_state": "FINISH"})
	assertNoWritingCommands(t, &f.fakeLink)
}

func TestBatchProgressIsWeightedAcrossEveryFileAndRejectsConcurrentUpload(t *testing.T) {
	f := &batchLink{blockAt: 2, entered: make(chan int, 3), unblock: make(chan struct{})}
	m := batchManager(t, f)
	if _, err := m.UploadPages("doc", "31", threePages); err != nil {
		t.Fatal(err)
	}
	<-f.entered
	<-f.entered
	s := m.State()
	if s.Page != 3 || s.UploadedPages != 1 || s.PageOrdinal != 2 || s.Progress != 46 {
		t.Fatal(s)
	}
	if _, err := m.UploadPages("doc", "32", threePages); err == nil {
		t.Fatal("concurrent upload accepted")
	}
	close(f.unblock)
	awaitPhase(t, m, "uploaded")
	assertNoWritingCommands(t, &f.fakeLink)
}

func TestBatchFailureStopsRemainingTransfers(t *testing.T) {
	f := &batchLink{failAt: 2}
	m := batchManager(t, f)
	if _, err := m.UploadPages("doc", "31", threePages); err != nil {
		t.Fatal(err)
	}
	awaitPhase(t, m, "failed")
	f.mu.Lock()
	if len(f.files) != 2 {
		t.Fatal("continued after failure")
	}
	f.mu.Unlock()
	if m.State().UploadedPages != 1 || m.State().JobID != "" || !strings.Contains(m.State().Message, "3") {
		t.Fatal(m.State())
	}
	assertNoWritingCommands(t, &f.fakeLink)
}

func TestBatchCancellationDisconnectAndHardwareActivityStopRemainingTransfers(t *testing.T) {
	for _, action := range []string{"cancel", "disconnect", "external"} {
		t.Run(action, func(t *testing.T) {
			f := &batchLink{blockAt: 2, entered: make(chan int, 3), unblock: make(chan struct{}), finished: make(chan int, 3)}
			m := batchManager(t, f)
			if _, err := m.UploadPages("doc", "31", threePages); err != nil {
				t.Fatal(err)
			}
			<-f.entered
			<-f.entered
			phase := "idle"
			switch action {
			case "cancel":
				if err := m.CancelUpload(); err != nil {
					t.Fatal(err)
				}
			case "disconnect":
				m.Close()
				phase = "disconnected"
			case "external":
				m.Observe(map[string]any{"gcode_state": "RUNNING"})
				phase = "external"
			}
			close(f.unblock)
			<-f.finished
			<-f.finished
			if m.State().JobID != "" || m.State().Phase != phase || m.State().UploadedPages != 1 {
				t.Fatal(m.State())
			}
			f.mu.Lock()
			if len(f.files) != 2 {
				t.Fatal("transferred remaining pages after interruption")
			}
			f.mu.Unlock()
			assertNoWritingCommands(t, &f.fakeLink)
		})
	}
}

func TestBatchRejectsInvalidInputBeforeTransferringAnything(t *testing.T) {
	f := &batchLink{}
	m := batchManager(t, f)
	for _, name := range []string{"", "../31", "31\n", "a/b", "a\\b", strings.Repeat("a", 65)} {
		if _, err := m.UploadPages("doc", name, threePages); err == nil {
			t.Fatalf("invalid name accepted: %q", name)
		}
	}
	for _, pages := range [][]PageUpload{nil, {{1, nil}}, {{0, []byte("x")}}, {{2, []byte("x")}, {1, []byte("x")}}} {
		if _, err := m.UploadPages("doc", "31", pages); err == nil {
			t.Fatal("invalid pages accepted")
		}
	}
	if len(f.files) != 0 {
		t.Fatal("invalid input initiated a transfer")
	}
}

func TestSpecificPageUsesCustomFilenameAndAllowsAnotherUpload(t *testing.T) {
	f := &batchLink{}
	m := batchManager(t, f)
	for _, prefix := range []string{"实验 31", "32"} {
		if _, err := m.UploadPages("doc", prefix, []PageUpload{{Page: 4, Data: []byte("fourth")}}); err != nil {
			t.Fatal(err)
		}
		awaitPhase(t, m, "uploaded")
	}
	f.mu.Lock()
	if len(f.files) != 2 || f.files[0].name != "实验 31-page4.gcode" || f.files[1].name != "32-page4.gcode" {
		t.Fatal(f.files)
	}
	f.mu.Unlock()
	assertNoWritingCommands(t, &f.fakeLink)
}
