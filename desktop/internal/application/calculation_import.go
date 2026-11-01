package application

import (
	"bambuscribe/desktop/internal/calculation"
	"bambuscribe/desktop/internal/calculationimport"
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/tableimport"
	"context"
	"fmt"
	"sync"
	"time"
)

type calculationState struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	generation uint64
}

func (a *App) RecognizeCalculations(s calculationimport.Sources) (calculation.Plan, error) {
	a.calculations.mu.Lock()
	if a.calculations.cancel != nil {
		a.calculations.mu.Unlock()
		return calculation.Plan{}, fmt.Errorf("正在识别数据，请等待或取消")
	}
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithTimeout(base, 5*time.Minute)
	a.calculations.generation++
	id := a.calculations.generation
	a.calculations.cancel = cancel
	a.calculations.mu.Unlock()
	defer func() {
		cancel()
		a.calculations.mu.Lock()
		if a.calculations.generation == id {
			a.calculations.cancel = nil
		}
		a.calculations.mu.Unlock()
	}()
	c, err := tableimport.LoadConfig(a.agentDir())
	if err != nil {
		return calculation.Plan{}, fmt.Errorf("模型配置无法读取，请检查模型连接")
	}
	return calculationimport.Recognize(ctx, c, s)
}
func (a *App) CancelCalculationRecognition() {
	a.calculations.mu.Lock()
	defer a.calculations.mu.Unlock()
	if a.calculations.cancel != nil {
		a.calculations.cancel()
	}
}

type CalculationPreview struct {
	calculation.Result
	Document *engine.Document `json:"document"`
}

func (a *App) PreviewCalculations(plan calculation.Plan, settings engine.LayoutSettings) (CalculationPreview, error) {
	var out CalculationPreview
	if a.fontError != nil {
		return out, a.fontError
	}
	result, err := plan.Calculate()
	if err != nil {
		return out, err
	}
	out.Result = result
	out.Document, err = a.font.Layout(result.Text, settings)
	return out, err
}
