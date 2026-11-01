package application

import (
	"bambuscribe/desktop/internal/liveeditor"
	"bambuscribe/desktop/internal/project"
	"bambuscribe/desktop/internal/tableimport"
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type importState struct {
	mu              sync.Mutex
	dir             string
	server          *liveeditor.Server
	nextRecognition uint64
	recognitions    map[uint64]context.CancelFunc
}

func (a *App) startTableImport(dir string) {
	a.imports.dir = dir
	server, err := liveeditor.Start(dir, func(req liveeditor.Request) { runtime.EventsEmit(a.ctx, "editor:request", req) })
	if err != nil {
		a.warning += " 当前文档 CLI 接口启动失败。"
		return
	}
	a.imports.server = server
}
func (a *App) AgentConfig() (tableimport.Config, error) {
	c, err := tableimport.LoadConfig(a.agentDir())
	return tableimport.Redacted(c), err
}
func (a *App) SaveAgentConfig(c tableimport.Config) error {
	a.imports.mu.Lock()
	defer a.imports.mu.Unlock()
	return tableimport.SaveConfig(a.agentDir(), c)
}
func (a *App) agentDir() string {
	if a.imports.dir != "" {
		return a.imports.dir
	}
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "BambuScribeWrite")
}
func (a *App) RecognizeTables(data, instruction string) (tableimport.Extraction, error) {
	a.imports.mu.Lock()
	if len(a.imports.recognitions) >= 3 {
		a.imports.mu.Unlock()
		return tableimport.Extraction{}, fmt.Errorf("最多同时识别 3 张图片，请等待或取消后重试")
	}
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithTimeout(base, 3*time.Minute)
	if a.imports.recognitions == nil {
		a.imports.recognitions = make(map[uint64]context.CancelFunc)
	}
	a.imports.nextRecognition++
	id := a.imports.nextRecognition
	a.imports.recognitions[id] = cancel
	c, err := tableimport.LoadConfig(a.agentDir())
	a.imports.mu.Unlock()
	defer func() { cancel(); a.imports.mu.Lock(); delete(a.imports.recognitions, id); a.imports.mu.Unlock() }()
	if err != nil {
		return tableimport.Extraction{}, fmt.Errorf("模型配置无法读取，请检查模型连接")
	}
	return tableimport.Recognize(ctx, c, data, instruction)
}
func (a *App) CancelTableRecognition() {
	a.imports.mu.Lock()
	defer a.imports.mu.Unlock()
	for _, cancel := range a.imports.recognitions {
		cancel()
	}
}
func (a *App) PlanTableImport(p project.Project, e tableimport.Extraction, where tableimport.Placement) (tableimport.Plan, error) {
	if a.fontError != nil {
		return tableimport.Plan{}, a.fontError
	}
	return tableimport.Build(a.font, p, e, where)
}
func (a *App) SetEditorReady(ready bool) {
	a.mu.Lock()
	a.editorReady = ready
	a.mu.Unlock()
	if a.imports.server != nil {
		a.imports.server.SetReady(ready)
	}
}
func (a *App) CompleteEditorRequest(id string, reply liveeditor.Reply) {
	if a.imports.server != nil {
		a.imports.server.Complete(id, reply)
	}
}

func (a *App) EditorRequestActive(id string) bool {
	return a.imports.server != nil && a.imports.server.Pending(id)
}
