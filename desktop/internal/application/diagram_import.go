package application

import (
	"bambuscribe/desktop/internal/diagram"
	"bambuscribe/desktop/internal/diagramagent"
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/tableimport"
	"context"
	"fmt"
	"sync"
	"time"
)

type diagramImportState struct {
	mu           sync.Mutex
	recognitions map[uint64]context.CancelFunc
	next         uint64
}

func (a *App) RecognizeDiagram(data, instruction string) (*diagram.Scene, error) {
	a.diagrams.mu.Lock()
	if len(a.diagrams.recognitions) >= 3 {
		a.diagrams.mu.Unlock()
		return nil, fmt.Errorf("最多同时识别 3 张图片，请等待或取消后重试")
	}
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithTimeout(base, 5*time.Minute)
	if a.diagrams.recognitions == nil {
		a.diagrams.recognitions = make(map[uint64]context.CancelFunc)
	}
	a.diagrams.next++
	id := a.diagrams.next
	a.diagrams.recognitions[id] = cancel
	a.diagrams.mu.Unlock()
	defer func() { cancel(); a.diagrams.mu.Lock(); delete(a.diagrams.recognitions, id); a.diagrams.mu.Unlock() }()
	a.imports.mu.Lock()
	c, err := tableimport.LoadConfig(a.agentDir())
	a.imports.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("模型配置无法读取，请检查模型连接")
	}
	if a.fontError != nil {
		return nil, a.fontError
	}
	layout := engine.DefaultLayout()
	if a.store != nil {
		layout = a.store.Snapshot().Preferences.Layout
	}
	font, err := a.font.ForLayout(layout)
	if err != nil {
		return nil, err
	}
	scene, err := diagramagent.Recognize(ctx, c, data, instruction, font.DiagramTypography(layout))
	if err != nil {
		return nil, err
	}
	if err = font.ValidateDiagram(scene); err != nil {
		return nil, err
	}
	return scene, nil
}

func (a *App) CancelDiagramRecognition() {
	a.diagrams.mu.Lock()
	defer a.diagrams.mu.Unlock()
	for _, cancel := range a.diagrams.recognitions {
		cancel()
	}
}
