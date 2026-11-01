package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"bambuscribe/desktop/internal/storage"
	"context"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func preferencesWithProject(p storage.Preferences, document project.Project) storage.Preferences {
	p.Calculations = document.Calculations
	p.Text, p.Layout, p.Writing = document.Text, document.Layout, document.Writing
	p.Images, p.Tables, p.Erasures = document.Images, document.Tables, document.Erasures
	return p
}

func projectFromPreferences(p storage.Preferences) project.Project {
	return project.Project{Calculations: p.Calculations, Version: project.Version, Text: p.Text, Layout: p.Layout, Writing: p.Writing, Images: p.Images, Tables: p.Tables, Erasures: p.Erasures}
}

func (a *App) workspaceSnapshot() (storage.Data, string, string) {
	a.persistenceMu.Lock()
	defer a.persistenceMu.Unlock()
	d := storage.Data{Preferences: storage.Defaults()}
	if a.store != nil {
		d = a.store.Snapshot()
	}
	path, warning := a.projectPath(), a.warning
	if path != "" {
		data, err := os.ReadFile(path)
		var document project.Project
		if err == nil {
			document, err = project.Decode(data)
		}
		if err != nil {
			// Never bind an empty fallback document to a file we could not read.
			warning += fmt.Sprintf(" 无法打开上次的项目 %s：%v。请重新打开项目文件。", path, err)
			a.setProjectPath("")
			path = ""
		} else {
			d.Preferences = preferencesWithProject(d.Preferences, document)
		}
	}
	return d, path, warning
}

// The file and its workspace reference are changed under persistenceMu. Only
// after both succeed does a later autosave start targeting the new location.
func (a *App) saveProjectAt(p project.Project, data []byte, path string) (string, error) {
	if err := storage.WriteAtomic(path, data, 0644); err != nil {
		return "", fmt.Errorf("保存失败：%w", err)
	}
	if err := a.rememberProject(p, path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) rememberProject(p project.Project, path string) error {
	if a.store != nil {
		preferences := preferencesWithProject(a.store.Snapshot().Preferences, p)
		if err := a.store.SaveWorkspace(preferences, path); err != nil {
			return fmt.Errorf("无法记住项目位置，请重试保存：%w", err)
		}
	}
	a.setProjectPath(path)
	return nil
}

// SavePreferences persists a new document as a draft, or writes a named
// document to its latest successful .ccf path. The expected path rejects a
// delayed save from a document that has since been replaced.
func (a *App) SavePreferences(p storage.Preferences, expectedPath string) error {
	a.persistenceMu.Lock()
	defer a.persistenceMu.Unlock()
	if a.store == nil {
		return os.ErrPermission
	}
	if expectedPath != a.projectPath() {
		return fmt.Errorf("项目已切换，请重试保存")
	}
	if err := p.Layout.Validate(); err != nil {
		return err
	}
	if err := engine.ValidateTables(p.Tables); err != nil {
		return err
	}
	if err := engine.ValidateErasures(p.Erasures); err != nil {
		return err
	}
	if expectedPath != "" {
		data, err := project.Encode(projectFromPreferences(p))
		if err != nil {
			return err
		}
		if err = storage.WriteAtomic(expectedPath, data, 0644); err != nil {
			return fmt.Errorf("项目自动保存失败，请检查文件位置或使用另存为：%w", err)
		}
	}
	if err := a.store.SaveWorkspace(p, expectedPath); err != nil {
		return err
	}
	a.mu.Lock()
	a.menuState.Lang = p.Language
	a.mu.Unlock()
	return nil
}

// Cancel the native close once so the editor can commit inline inputs and
// finish the debounce/in-flight save. A failed save leaves the window open.
func (a *App) BeforeClose(ctx context.Context) bool {
	a.mu.Lock()
	blocked := a.editorReady && !a.closeAllowed
	a.mu.Unlock()
	if blocked {
		runtime.EventsEmit(ctx, "workspace:close")
	}
	return blocked
}

func (a *App) CompleteClose() {
	a.mu.Lock()
	a.closeAllowed = true
	ctx := a.ctx
	a.mu.Unlock()
	runtime.Quit(ctx)
}
