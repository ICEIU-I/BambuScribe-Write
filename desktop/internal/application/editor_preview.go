package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/preview"
)

func (a *App) PreviewEditor(text string, settings engine.LayoutSettings, images []engine.ImageAsset, tables []engine.TableAsset, erasures []engine.PageErasure) (*preview.Document, error) {
	doc, err := a.PreviewWithErasures(text, settings, images, tables, erasures)
	if err != nil {
		return nil, err
	}
	return preview.Pack(doc), nil
}
