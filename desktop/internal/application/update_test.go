package application

import (
	"bambuscribe/desktop/internal/updater"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFailedCheckPreservesKnownUpdate(t *testing.T) {
	fail := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()
	known := &updater.Release{Tag: "v0.1.4"}
	a := &App{}
	a.updates.client = &updater.Client{HTTP: server.Client(), Endpoint: server.URL}
	a.updates.release = known
	if _, err := a.CheckUpdate(); err == nil {
		t.Fatal("failed check should return an error")
	}
	if a.updates.release != known {
		t.Fatal("failed background check discarded the previously verified update")
	}
	fail = false
	if info, err := a.CheckUpdate(); err != nil || info.Available {
		t.Fatalf("successful empty check = %+v, %v", info, err)
	}
	if a.updates.release != nil {
		t.Fatal("successful check retained a withdrawn update")
	}
}

func TestUpdateOperationsSerializeAndCanBeCancelled(t *testing.T) {
	a := &App{}
	ctx, _, err := a.beginUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.beginUpdate(); err == nil {
		t.Fatal("overlapping update accepted")
	}
	a.CancelUpdate()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("cancel did not reach downloader")
	}
	a.endUpdate()
	if _, _, err := a.beginUpdate(); err != nil {
		t.Fatal("could not retry after cancellation")
	}
	a.endUpdate()
	if err := a.InstallUpdate(); err == nil {
		t.Fatal("install accepted without a checked release")
	}
	if updateError(updater.ErrChecksum).Error() == "" {
		t.Fatal("empty error message")
	}
}
