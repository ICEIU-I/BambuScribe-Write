package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/output"
	"bambuscribe/desktop/internal/printer"
)

func uploadPages(d *engine.Document, w engine.WritingSettings, indices ...int) ([]printer.PageUpload, error) {
	return output.PreparePages(d, w, indices...)
}

func (a *App) UploadPages(id, prefix string, indices []int, w engine.WritingSettings) (string, error) {
	return a.UploadDevicePages(a.Printers().SelectedDeviceID, id, prefix, indices, w)
}

func (a *App) UploadDevicePages(deviceID, id, prefix string, indices []int, w engine.WritingSettings) (string, error) {
	if err := printer.ValidateUploadPrefix(prefix); err != nil {
		return "", err
	}
	d, err := a.document(id)
	if err != nil {
		return "", err
	}
	// Validate every page before transferring any file to the printer.
	pages, err := uploadPages(d, w, indices...)
	if err != nil {
		return "", err
	}
	return a.fleet.UploadPages(deviceID, id, prefix, pages)
}
