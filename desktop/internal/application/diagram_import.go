package application

import (
	"bambuscribe/desktop/internal/diagram"
	"bambuscribe/desktop/internal/diagramagent"
	"bambuscribe/desktop/internal/tableimport"
	"context"
	"fmt"
	"sync"
	"time"
)

type diagramImportState struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (a *App) RecognizeDiagram(data, instruction string) (*diagram.Scene, error) {
	a.diagrams.mu.Lock()
	if a.diagrams.cancel != nil {
		a.diagrams.mu.Unlock()
		return nil, fmt.Errorf("图像正在识别，请等待或取消后重试")
	}
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithTimeout(base, 5*time.Minute)
	a.diagrams.cancel = cancel
	a.diagrams.mu.Unlock()
	defer func() { cancel(); a.diagrams.mu.Lock(); a.diagrams.cancel = nil; a.diagrams.mu.Unlock() }()
	a.imports.mu.Lock()
	c, err := tableimport.LoadConfig(a.agentDir())
	a.imports.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("模型配置无法读取，请检查模型连接")
	}
	scene, err := diagramagent.Recognize(ctx, c, data, instruction)
	if err != nil {
		return nil, err
	}
	if a.fontError != nil {
		return nil, a.fontError
	}
	if err = a.font.ValidateDiagram(scene); err != nil {
		return nil, err
	}
	return scene, nil
}

func (a *App) CancelDiagramRecognition() {
	a.diagrams.mu.Lock()
	defer a.diagrams.mu.Unlock()
	if a.diagrams.cancel != nil {
		a.diagrams.cancel()
	}
}
