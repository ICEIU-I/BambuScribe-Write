package printer

import (
	"bambuscribe/desktop/internal/i18n"
	"context"
	"fmt"
	"sync"
)

type State struct {
	DeviceID      string `json:"deviceId,omitempty"`
	Connected     bool   `json:"connected"`
	Phase         string `json:"phase"`
	Message       string `json:"message"`
	Progress      int    `json:"progress"`
	JobID         string `json:"jobId"`
	DocumentID    string `json:"documentId"`
	Page          int    `json:"page"`
	TotalPages    int    `json:"totalPages"`
	PageOrdinal   int    `json:"pageOrdinal"`
	UploadedPages int    `json:"uploadedPages"`
	DeviceState   string `json:"deviceState"`

	// The message is stored by key and rendered on read, so it follows the
	// active language without the call sites knowing about it.
	messageKey  string
	messageArgs []any
}
type Manager struct {
	mu           sync.Mutex
	link         Link
	state        State
	emit         func(State)
	cancelUpload context.CancelFunc
}

func NewManager(link Link, emit func(State)) *Manager {
	if emit == nil {
		emit = func(State) {}
	}
	return &Manager{link: link, emit: emit, state: State{Phase: "disconnected"}}
}
func (m *Manager) State() State { m.mu.Lock(); defer m.mu.Unlock(); return m.renderLocked() }
func (m *Manager) notify()      { m.emit(m.State()) }

// SetLanguage re-emits the state so a language change reaches the interface.
// The message is rendered from its key on every read, so nothing is cached in
// the old language.
func (m *Manager) SetLanguage() { m.notify() }

// setMessageLocked records a message by key; callers hold the lock.
func (m *Manager) setMessageLocked(key string, args ...any) {
	m.state.messageKey = key
	m.state.messageArgs = args
}

// renderLocked fills in the message for the active language. A state whose
// message was set directly (a raw error) is passed through untouched.
func (m *Manager) renderLocked() State {
	s := m.state
	if s.messageKey != "" {
		s.Message = i18n.T(s.messageKey, s.messageArgs...)
	}
	return s
}

func idleDevice(s string) bool { return s == "IDLE" || s == "FINISH" || s == "FAILED" }
func (m *Manager) readyLocked() error {
	if !m.state.Connected {
		return fmt.Errorf("请先连接打印机")
	}
	if !idleDevice(m.state.DeviceState) {
		return fmt.Errorf("请等待打印机上报空闲状态")
	}
	switch m.state.Phase {
	case "idle", "failed", "uploaded":
		return nil
	}
	return fmt.Errorf("当前正在上传，请等待完成或取消上传")
}
