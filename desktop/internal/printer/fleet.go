package printer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type deviceSession struct {
	op             sync.Mutex
	mu             sync.Mutex
	manager        *Manager
	detecting      bool
	detectionError string
	deleted        bool
}

// Fleet keeps device operations independent: a slow connection or upload on
// one A1 never holds another A1's session lock.
type Fleet struct {
	mu       sync.Mutex
	sessions map[string]*deviceSession
	newLink  func() Link
	discover func(context.Context, string) (string, error)
	emit     func(State)
}

func NewFleet(newLink func() Link, discover func(context.Context, string) (string, error), emit func(State)) *Fleet {
	if emit == nil {
		emit = func(State) {}
	}
	return &Fleet{sessions: map[string]*deviceSession{}, newLink: newLink, discover: discover, emit: emit}
}

func (f *Fleet) Add(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sessions[id] != nil {
		return
	}
	s := &deviceSession{}
	s.manager = NewManager(f.newLink(), func(state State) {
		state.DeviceID = id
		f.emit(state)
	})
	f.sessions[id] = s
}

func (f *Fleet) session(id string) (*deviceSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.sessions[id]
	if s == nil {
		return nil, fmt.Errorf("实例已删除，请重新选择")
	}
	return s, nil
}

func sessionState(id string, s *deviceSession) State {
	state := s.manager.State()
	s.mu.Lock()
	defer s.mu.Unlock()
	state.DeviceID = id
	if s.detecting {
		state.Phase, state.Message = "connecting", ""
	}
	if s.detectionError != "" {
		state.Message = s.detectionError
	}
	return state
}

func (f *Fleet) State(id string) State {
	s, err := f.session(id)
	if err != nil {
		return State{DeviceID: id, Phase: "disconnected"}
	}
	return sessionState(id, s)
}

func (f *Fleet) Connect(ctx context.Context, id string, c Config) error {
	s, err := f.session(id)
	if err != nil {
		return err
	}
	s.op.Lock()
	defer s.op.Unlock()
	if s.deleted {
		return fmt.Errorf("实例已删除，请重新选择")
	}
	if s.manager.State().Connected {
		return fmt.Errorf("请先断开此实例的连接")
	}
	s.mu.Lock()
	s.detecting, s.detectionError = true, ""
	s.mu.Unlock()
	f.emit(sessionState(id, s))
	detectCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	c.Serial, err = f.discover(detectCtx, c.IP)
	cancel()
	if err == nil {
		err = c.Validate()
	}
	s.mu.Lock()
	s.detecting = false
	if err != nil {
		s.detectionError = err.Error()
	}
	s.mu.Unlock()
	if err != nil {
		f.emit(sessionState(id, s))
		return err
	}
	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return s.manager.Connect(connectCtx, c)
}

func (f *Fleet) Disconnect(id string) error {
	s, err := f.session(id)
	if err != nil {
		return err
	}
	s.op.Lock()
	defer s.op.Unlock()
	s.mu.Lock()
	s.detectionError = ""
	s.mu.Unlock()
	s.manager.Close()
	return nil
}

func (f *Fleet) Remove(id string) {
	s, err := f.session(id)
	if err != nil {
		return
	}
	s.op.Lock()
	defer s.op.Unlock()
	s.deleted = true
	s.manager.Close()
	f.mu.Lock()
	delete(f.sessions, id)
	f.mu.Unlock()
}

func (f *Fleet) UploadPages(id, documentID, prefix string, pages []PageUpload) (string, error) {
	s, err := f.session(id)
	if err != nil {
		return "", err
	}
	s.op.Lock()
	defer s.op.Unlock()
	if s.deleted {
		return "", fmt.Errorf("实例已删除，请重新选择")
	}
	return s.manager.UploadPages(documentID, prefix, pages)
}

func (f *Fleet) CancelUpload(id string) error {
	s, err := f.session(id)
	if err != nil {
		return err
	}
	return s.manager.CancelUpload()
}

func (f *Fleet) Close() {
	f.mu.Lock()
	ids := make([]string, 0, len(f.sessions))
	for id := range f.sessions {
		ids = append(ids, id)
	}
	f.mu.Unlock()
	for _, id := range ids {
		f.Remove(id)
	}
}

func (f *Fleet) SetLanguage() {
	f.mu.Lock()
	managers := make([]*Manager, 0, len(f.sessions))
	for _, s := range f.sessions {
		managers = append(managers, s.manager)
	}
	f.mu.Unlock()
	for _, m := range managers {
		m.SetLanguage()
	}
}
