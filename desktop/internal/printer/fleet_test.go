package printer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fleetLink struct {
	batchLink
	configs []Config
	closed  int
}

func (l *fleetLink) Connect(_ context.Context, c Config, report func(map[string]any), _ func(error)) error {
	l.mu.Lock()
	l.configs = append(l.configs, c)
	l.mu.Unlock()
	report(map[string]any{"gcode_state": "IDLE"})
	return nil
}
func (l *fleetLink) Close() { l.mu.Lock(); l.closed++; l.mu.Unlock() }

func TestFleetConnectionsUploadsAndDeletionStayIndependent(t *testing.T) {
	links := []*fleetLink{
		{batchLink: batchLink{blockAt: 1, entered: make(chan int, 3), unblock: make(chan struct{})}},
		{},
	}
	index := 0
	var mu sync.Mutex
	detections := map[string]int{}
	entered, release := make(chan struct{}), make(chan struct{})
	f := NewFleet(func() Link { l := links[index]; index++; return l }, func(ctx context.Context, ip string) (string, error) {
		mu.Lock()
		detections[ip]++
		mu.Unlock()
		if ip == "127.0.0.1" {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		return "DETECTED123", nil
	}, nil)
	f.Add("one")
	f.Add("two")
	defer f.Close()
	first := make(chan error, 1)
	go func() {
		first <- f.Connect(context.Background(), "one", Config{IP: "127.0.0.1", AccessCode: "one", Serial: "STALE123"})
	}()
	<-entered
	if f.State("one").Phase != "connecting" {
		t.Fatal("detection phase missing")
	}
	// A second connection must complete before the first discovery is released.
	second := make(chan error, 1)
	go func() { second <- f.Connect(context.Background(), "two", Config{IP: "127.0.0.2", AccessCode: "two"}) }()
	select {
	case err := <-second:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("second printer blocked by first")
	}
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if !f.State("one").Connected || !f.State("two").Connected || f.State("two").DeviceID != "two" {
		t.Fatal("connections mixed")
	}
	if links[0].configs[0].Serial != "DETECTED123" {
		t.Fatal("stale serial used")
	}
	if _, err := f.UploadPages("one", "doc", "31", threePages); err != nil {
		t.Fatal(err)
	}
	<-links[0].entered
	if _, err := f.UploadPages("two", "doc", "32", threePages); err != nil {
		t.Fatal(err)
	}
	s, _ := f.session("two")
	awaitPhase(t, s.manager, "uploaded")
	if f.State("one").Phase != "uploading" {
		t.Fatal("second upload changed first")
	}
	if err := f.CancelUpload("one"); err != nil {
		t.Fatal(err)
	}
	if f.State("two").UploadedPages != 3 {
		t.Fatal("cancel affected another printer")
	}
	close(links[0].unblock)
	f.Remove("one")
	if !f.State("two").Connected || links[1].closed != 0 {
		t.Fatal("delete disconnected another printer")
	}
	if err := f.Connect(context.Background(), "one", Config{}); err == nil {
		t.Fatal("deleted device reconnected")
	}
	assertNoWritingCommands(t, &links[0].fakeLink)
	assertNoWritingCommands(t, &links[1].fakeLink)
}

func TestFleetDiscoveryFailureNeverReusesCachedSerial(t *testing.T) {
	l := &fleetLink{}
	fail := true
	f := NewFleet(func() Link { return l }, func(context.Context, string) (string, error) {
		if fail {
			return "", errors.New("offline")
		}
		return "FRESH12345", nil
	}, nil)
	f.Add("one")
	defer f.Close()
	c := Config{IP: "127.0.0.1", AccessCode: "code", Serial: "CACHED123"}
	if err := f.Connect(context.Background(), "one", c); err == nil || len(l.configs) != 0 || f.State("one").Message != "offline" {
		t.Fatal("discovery failure bypassed")
	}
	fail = false
	if err := f.Connect(context.Background(), "one", c); err != nil {
		t.Fatal(err)
	}
	if l.configs[0].Serial != "FRESH12345" || !f.State("one").Connected {
		t.Fatal("retry did not discover current printer")
	}
}
