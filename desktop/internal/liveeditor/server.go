// Package liveeditor exposes only current-document snapshot and table import
// over authenticated loopback. The webview owns editing and undo history.
package liveeditor

import (
	"bambuscribe/desktop/internal/storage"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Session struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}
type Request struct {
	ID     string          `json:"id"`
	Action string          `json:"action"`
	Params json.RawMessage `json:"params"`
}
type Reply struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}
type Server struct {
	http     *http.Server
	session  Session
	path     string
	mu       sync.Mutex
	pending  map[string]chan Reply
	dispatch func(Request)
	ready    bool
}

func Start(dir string, dispatch func(Request)) (*Server, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		ln.Close()
		return nil, err
	}
	s := &Server{session: Session{"http://" + ln.Addr().String(), hex.EncodeToString(b)}, path: filepath.Join(dir, "editor-session.json"), pending: map[string]chan Reply{}, dispatch: dispatch}
	s.http = &http.Server{Handler: http.HandlerFunc(s.handle), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 60 * time.Second}
	raw, _ := json.Marshal(s.session)
	if err = os.MkdirAll(dir, 0700); err == nil {
		err = storage.WriteAtomic(s.path, raw, 0600)
	}
	if err != nil {
		ln.Close()
		return nil, err
	}
	go s.http.Serve(ln)
	return s, nil
}
func (s *Server) SetReady(v bool) { s.mu.Lock(); s.ready = v; s.mu.Unlock() }
func (s *Server) Pending(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.pending[id]
	return ok
}
func (s *Server) Complete(id string, reply Reply) {
	s.mu.Lock()
	ch := s.pending[id]
	s.mu.Unlock()
	if ch != nil {
		select {
		case ch <- reply:
		default:
		}
	}
}
func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fail := func(code int, message string) { w.WriteHeader(code); json.NewEncoder(w).Encode(Reply{Error: message}) }
	if r.Method != "POST" || r.URL.Path != "/editor" {
		fail(404, "未知编辑请求")
		return
	}
	if r.Header.Get("Origin") != "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.session.Token)) != 1 {
		fail(403, "编辑会话认证失败")
		return
	}
	var req Request
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
	dec.DisallowUnknownFields()
	if dec.Decode(&req) != nil || dec.Decode(new(any)) != io.EOF || (req.Action != "snapshot" && req.Action != "import") {
		fail(400, "仅支持 snapshot 和 import")
		return
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		fail(500, "无法创建编辑请求")
		return
	}
	req.ID = hex.EncodeToString(b)
	ch := make(chan Reply, 1)
	s.mu.Lock()
	if !s.ready || len(s.pending) >= 8 {
		s.mu.Unlock()
		fail(503, "客户端编辑器尚未就绪，请稍后重试")
		return
	}
	s.pending[req.ID] = ch
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.pending, req.ID); s.mu.Unlock() }()
	s.dispatch(req)
	timer := time.NewTimer(45 * time.Second)
	defer timer.Stop()
	select {
	case reply := <-ch:
		if reply.Error != "" {
			w.WriteHeader(400)
		}
		json.NewEncoder(w).Encode(reply)
	case <-r.Context().Done():
		return
	case <-timer.C:
		fail(504, "编辑请求超时，请检查客户端内容；不要自动重复导入")
	}
}
func (s *Server) Close() {
	s.SetReady(false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.http.Shutdown(ctx)
	b, _ := os.ReadFile(s.path)
	var current Session
	if json.Unmarshal(b, &current) == nil && current.Token == s.session.Token {
		os.Remove(s.path)
	}
}
func Call(ctx context.Context, path, action string, params json.RawMessage) (json.RawMessage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("没有运行中的客户端编辑会话，请打开客户端或指定 --session")
	}
	var session Session
	if json.Unmarshal(b, &session) != nil {
		return nil, fmt.Errorf("编辑会话文件无效，请重启客户端")
	}
	// Session files cannot redirect the CLI's bearer token to a remote host.
	u, err := url.Parse(session.URL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("编辑会话必须使用本机地址")
	}

	if params == nil {
		params = json.RawMessage(`{}`)
	}
	raw, err := json.Marshal(Request{Action: action, Params: params})
	if err != nil {
		return nil, fmt.Errorf("导入参数 JSON 无效")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", session.URL+"/editor", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+session.Token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 50 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("客户端编辑会话连接失败；请检查客户端，导入超时后先检查文档再重试")
	}
	defer res.Body.Close()
	var reply Reply
	if err = json.NewDecoder(io.LimitReader(res.Body, 32<<20)).Decode(&reply); err != nil {
		return nil, fmt.Errorf("编辑会话响应无效")
	}
	if reply.Error != "" {
		return nil, fmt.Errorf("%s", reply.Error)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("编辑会话返回 HTTP %d", res.StatusCode)
	}
	return reply.Result, nil
}
