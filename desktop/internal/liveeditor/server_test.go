package liveeditor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestAuthenticatedSessionRoundtrip(t *testing.T) {
	dir := t.TempDir()
	var server *Server
	var err error
	server, err = Start(dir, func(req Request) { server.Complete(req.ID, Reply{Result: json.RawMessage(`{"revision":7}`)}) })
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	server.SetReady(true)
	result, err := Call(context.Background(), filepath.Join(dir, "editor-session.json"), "snapshot", nil)
	if err != nil || string(result) != `{"revision":7}` {
		t.Fatal(string(result), err)
	}
	for _, origin := range []string{"", "http://other.example"} {
		req, _ := http.NewRequest("POST", server.session.URL+"/editor", bytes.NewBufferString(`{"action":"import","params":{}}`))
		if origin != "" {
			req.Header.Set("Authorization", "Bearer "+server.session.Token)
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 403 {
			t.Fatal("unauthenticated request accepted")
		}
	}
	server.SetReady(false)
	if _, err = Call(context.Background(), server.path, "snapshot", nil); err == nil {
		t.Fatal("not-ready accepted")
	}
	info, _ := os.Stat(server.path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("session permissions")
	}
	server.Close()
	if _, err = os.Stat(server.path); !os.IsNotExist(err) {
		t.Fatal("stale session not removed")
	}
}
func TestSessionCannotSendTokenToRemoteHost(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.json")
	os.WriteFile(p, []byte(`{"url":"http://example.com:8080","token":"secret"}`), 0600)
	if _, err := Call(context.Background(), p, "snapshot", nil); err == nil {
		t.Fatal("accepted remote session")
	}
}
