package storage

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/printer"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu        sync.Mutex
	path      string
	data      Data
	loadError error
}

func Open(dir string) (*Store, error) {
	s := &Store{path: filepath.Join(dir, "settings.json"), data: Data{Version: 1, Preferences: Defaults(), Devices: []printer.Device{}}}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		s.loadError = err
		return s, err
	}
	var data Data
	if err = json.Unmarshal(b, &data); err != nil {
		s.loadError = err
		return s, err
	}
	if data.Version != 1 {
		s.loadError = fmt.Errorf("不支持的配置版本")
		return s, s.loadError
	}
	if data.Preferences.Theme != "dark" {
		data.Preferences.Theme = "light"
	}
	if data.Preferences.Layout.Align == "" {
		data.Preferences.Layout.Align = "left"
	}
	if data.Preferences.Writing.OriginMode == "" {
		data.Preferences.Writing = engine.DefaultWriting()
	}
	if data.Preferences.Layout.LineMode == "" {
		data.Preferences.Layout.LineMode = "fixed"
		if data.Preferences.Layout.LineHeight > 0 {
			data.Preferences.Layout.LineCount = max(1, int((data.Preferences.Layout.PaperHeight-data.Preferences.Layout.MarginTop-data.Preferences.Layout.MarginBottom)/data.Preferences.Layout.LineHeight))
		} else {
			data.Preferences.Layout.LineCount = 16
		}
	}
	if data.Preferences.Language == "" {
		data.Preferences.Language = "system"
	}
	if data.Preferences.Layout.TemplateName == "" {
		data.Preferences.Layout.TemplateName = "新建笔记"
	}
	normalizeDevices(&data)
	s.data = data
	return s, nil
}

func (s *Store) Snapshot() Data {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.data
	d.Devices = append([]printer.Device{}, d.Devices...)
	return d
}

func (s *Store) SavePreferences(p Preferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	if next.ProjectPath != "" {
		p = withoutDocument(p)
	}
	next.Preferences = p
	return s.save(next)
}

// SaveWorkspace keeps either a draft or a project reference, never both.
func (s *Store) SaveWorkspace(p Preferences, projectPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.ProjectPath = projectPath
	if projectPath != "" {
		p = withoutDocument(p)
	}
	next.Preferences = p
	return s.save(next)
}

func withoutDocument(p Preferences) Preferences {
	p.Text = ""
	p.Images = nil
	p.Tables = nil
	p.Erasures = nil
	p.Layout.PageLayouts = nil
	return p
}

func (s *Store) SavePrinter(c printer.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Printer = c
	next.Devices = append([]printer.Device{}, next.Devices...)
	if len(next.Devices) == 0 {
		next.Devices = []printer.Device{{ID: "primary", Name: "A1 1", Config: c}}
		next.SelectedDeviceID = "primary"
	} else {
		for i := range next.Devices {
			if next.Devices[i].ID == next.SelectedDeviceID {
				next.Devices[i].Config = c
			}
		}
	}
	return s.save(next)
}

func (s *Store) save(next Data) error {
	if s.loadError != nil {
		return fmt.Errorf("原配置读取失败，为保留数据暂不覆盖，请检查 settings.json")
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	if err = WriteAtomic(s.path, b, 0600); err != nil {
		return err
	}
	s.data = next
	return nil
}

func WriteAtomic(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".scribe-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
