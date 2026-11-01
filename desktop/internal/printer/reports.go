package printer

// Device reports only gate uploads. Writing is controlled on the A1 itself.
func (m *Manager) Observe(p map[string]any) {
	m.mu.Lock()
	if !m.state.Connected && m.state.Phase != "connecting" {
		m.mu.Unlock()
		return
	}
	state, _ := p["gcode_state"].(string)
	if state != "" {
		m.state.DeviceState = state
	}
	if state == "RUNNING" || state == "PAUSE" {
		if m.cancelUpload != nil {
			m.cancelUpload()
		}
		m.cancelUpload = nil
		m.state.JobID = ""
		m.state.Phase = "external"
		m.setMessageLocked("printer.deviceBusy")
	} else if idleDevice(state) && (m.state.Phase == "external" || m.state.Phase == "idle") {
		m.state.Phase = "idle"
		m.setMessageLocked("printer.deviceIdle")
	}
	m.mu.Unlock()
	m.notify()
}
