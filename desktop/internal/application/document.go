package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/storage"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) Preview(text string, settings engine.LayoutSettings) (*engine.Document, error) {
	return a.PreviewWithImages(text, settings, nil)
}

func (a *App) PreviewWithImages(text string, settings engine.LayoutSettings, images []engine.ImageAsset) (*engine.Document, error) {
	return a.PreviewWithObjects(text, settings, images, nil)
}

func (a *App) PreviewWithObjects(text string, settings engine.LayoutSettings, images []engine.ImageAsset, tables []engine.TableAsset) (*engine.Document, error) {
	return a.PreviewWithErasures(text, settings, images, tables, nil)
}

func (a *App) PreviewWithErasures(text string, settings engine.LayoutSettings, images []engine.ImageAsset, tables []engine.TableAsset, erasures []engine.PageErasure) (*engine.Document, error) {
	if a.fontError != nil {
		return nil, a.fontError
	}
	d, err := a.font.LayoutWithErasures(text, settings, images, tables, erasures)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	if _, ok := a.documents[d.ID]; !ok {
		a.documentOrder = append(a.documentOrder, d.ID)
	}
	a.documents[d.ID] = d
	for len(a.documentOrder) > 3 {
		delete(a.documents, a.documentOrder[0])
		a.documentOrder = a.documentOrder[1:]
	}
	a.mu.Unlock()
	return d, nil
}

func (a *App) document(id string) (*engine.Document, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	d := a.documents[id]
	if d == nil {
		return nil, fmt.Errorf("预览已更新，请等待当前页面排版完成")
	}
	return d, nil
}

func (a *App) ExportPage(id string, index int, w engine.WritingSettings) (string, error) {
	d, err := a.document(id)
	if err != nil {
		return "", err
	}
	data, err := engine.ExportPage(d, index, w)
	if err != nil {
		return "", err
	}
	return a.saveExport(fmt.Sprintf("page-%03d.gcode", index+1), "*.gcode", data)
}

func (a *App) ExportAll(id string, w engine.WritingSettings) (string, error) {
	d, err := a.document(id)
	if err != nil {
		return "", err
	}
	data, err := engine.ExportAll(d, w)
	if err != nil {
		return "", err
	}
	return a.saveExport("handwriting-pages.zip", "*.zip", data)
}

func (a *App) saveExport(filename, pattern string, data []byte) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "导出书写文件", DefaultFilename: filename, Filters: []runtime.FileFilter{{DisplayName: "书写文件", Pattern: pattern}}, CanCreateDirectories: true})
	if err != nil || path == "" {
		return "", err
	}
	ext := strings.TrimPrefix(pattern, "*")
	if !strings.HasSuffix(strings.ToLower(path), ext) {
		path += ext
	}
	if err = storage.WriteAtomic(path, data, 0644); err != nil {
		return "", fmt.Errorf("保存失败：%w", err)
	}
	return path, nil
}
