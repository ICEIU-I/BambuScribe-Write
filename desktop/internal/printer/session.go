package printer

import (
	"context"
	"fmt"
)

func (m *Manager) Connect(ctx context.Context, c Config) error {
	m.mu.Lock()
	if m.state.Connected || m.state.Phase == "connecting" {
		m.mu.Unlock()
		return fmt.Errorf("请先断开当前连接")
	}
	// The dot and the phase badge already say "connecting"; only errors and
	// real device states carry a message.
	m.state = State{Phase: "connecting"}
	m.mu.Unlock()
	m.notify()
	err := m.link.Connect(ctx, c, m.Observe, m.lost)
	m.mu.Lock()
	if err != nil {
		m.state = State{Phase: "disconnected", Message: err.Error()}
	} else {
		m.state.Connected = true
		m.state.Phase = "idle"
		m.setMessageLocked("printer.connected")
	}
	m.mu.Unlock()
	m.notify()
	if err == nil {
		err = m.link.Publish(map[string]any{"pushing": map[string]any{"sequence_id": "0", "command": "pushall"}})
	}
	return err
}
func (m *Manager) lost(err error) {
	m.mu.Lock()
	if m.cancelUpload != nil {
		m.cancelUpload()
	}
	m.cancelUpload = nil
	m.state.JobID = ""
	m.state.Connected = false
	m.state.Phase = "disconnected"
	if err != nil {
		m.setMessageLocked("printer.linkLost")
	} else {
		m.state.messageKey, m.state.Message, m.state.messageArgs = "", "", nil
	}
	m.mu.Unlock()
	m.notify()
}
func (m *Manager) Close() {
	m.mu.Lock()
	if m.cancelUpload != nil {
		m.cancelUpload()
	}
	m.mu.Unlock()
	m.link.Close()
	m.lost(nil)
}
