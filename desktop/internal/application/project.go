package application

import (
	"bambuscribe/desktop/internal/i18n"
	"bambuscribe/desktop/internal/project"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const projectFilter = "*" + project.Extension

// ProjectFile is an opened project together with the path it came from, so the
// editor can name the current file and save back to it without asking again.
type ProjectFile struct {
	Path    string          `json:"path"`
	Project project.Project `json:"project"`
}

// ConfirmDiscardProject asks what to do about unsaved edits before another
// project replaces them. It answers "save", "discard" or "cancel".
func (a *App) ConfirmDiscardProject() (string, error) {
	save, discard, cancel := i18n.T("app.saveAndContinue"), i18n.T("app.discard"), i18n.T("app.cancel")
	choice, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         i18n.T("app.unsavedTitle"),
		Message:       i18n.T("app.unsavedBody"),
		Buttons:       []string{save, discard, cancel},
		DefaultButton: save,
		CancelButton:  cancel,
	})
	if err != nil {
		return "", err
	}
	switch choice {
	case save:
		return "save", nil
	case discard:
		return "discard", nil
	default:
		return "cancel", nil
	}
}

// NewProject detaches the next document from the previously opened file.
// The frontend confirms unsaved edits before calling this method.
func (a *App) NewProject() error {
	a.persistenceMu.Lock()
	defer a.persistenceMu.Unlock()
	if a.store != nil {
		p := a.store.Snapshot().Preferences
		p.Text, p.Images, p.Tables, p.Erasures, p.Layout.PageLayouts = "", nil, nil, nil, nil
		if err := a.store.SaveWorkspace(p, ""); err != nil {
			return err
		}
	}
	a.setProjectPath("")
	return nil
}

// RevealProject shows a saved project in the system file manager.
func (a *App) RevealProject(path string) error {
	if path == "" {
		return fmt.Errorf("尚未保存为项目文件")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("文件已不存在，请重新保存")
	}
	if err := reveal(path); err != nil {
		return fmt.Errorf("无法打开所在目录：%w", err)
	}
	return nil
}

// SaveProject writes to the file the project was opened from, or asks for a
// location when it has none. An empty path means the user cancelled.
func (a *App) SaveProject(p project.Project) (string, error) { return a.saveProject(p, false) }

// SaveProjectAs always asks for a location.
func (a *App) SaveProjectAs(p project.Project) (string, error) { return a.saveProject(p, true) }

func (a *App) saveProject(p project.Project, choose bool) (string, error) {
	a.persistenceMu.Lock()
	defer a.persistenceMu.Unlock()
	data, err := project.Encode(p)
	if err != nil {
		return "", err
	}
	path := a.projectPath()
	if choose || path == "" {
		directory, err := defaultProjectDirectory()
		if err != nil {
			return "", err
		}
		if path, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:                "保存项目",
			DefaultDirectory:     directory,
			DefaultFilename:      defaultProjectFilename(p.Text),
			Filters:              []runtime.FileFilter{{DisplayName: "BambuScribe 项目", Pattern: projectFilter}},
			CanCreateDirectories: true,
		}); err != nil || path == "" {
			return "", err
		}
		if !strings.HasSuffix(strings.ToLower(path), project.Extension) {
			path += project.Extension
		}
	}
	return a.saveProjectAt(p, data, path)
}

// OpenProject asks for a project file and reads it. A nil result means the
// user cancelled; a read failure leaves the current project untouched.
func (a *App) OpenProject() (*ProjectFile, error) {
	a.persistenceMu.Lock()
	defer a.persistenceMu.Unlock()
	directory, err := defaultProjectDirectory()
	if err != nil {
		return nil, err
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "打开项目",
		DefaultDirectory: directory,
		Filters:          []runtime.FileFilter{{DisplayName: "BambuScribe 项目", Pattern: projectFilter}},
	})
	if err != nil || path == "" {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取失败：%w", err)
	}
	p, err := project.Decode(data)
	if err != nil {
		return nil, err
	}
	if err = a.rememberProject(p, path); err != nil {
		return nil, err
	}
	return &ProjectFile{Path: path, Project: p}, nil
}

func (a *App) projectPath() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.projectFile
}

func (a *App) setProjectPath(path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.projectFile = path
}
