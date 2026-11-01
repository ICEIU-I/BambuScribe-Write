package printer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type uploadFile struct {
	Name  string
	Page  int
	Bytes []byte
}

type PageUpload struct {
	Page int
	Data []byte
}

func ValidateUploadPrefix(prefix string) error {
	if strings.TrimSpace(prefix) != prefix || prefix == "" || utf8.RuneCountInString(prefix) > 64 || strings.ContainsAny(prefix, `/\:*?"<>|`) || strings.IndexFunc(prefix, unicode.IsControl) >= 0 {
		return fmt.Errorf("文件名前缀无效，请使用 1–64 个字符，且不要包含 / \\ : * ? \" < > |")
	}
	return nil
}

// UploadPages transfers every file in the batch without starting the printer.
func (m *Manager) UploadPages(docID, prefix string, pages []PageUpload) (string, error) {
	if err := ValidateUploadPrefix(prefix); err != nil {
		return "", err
	}
	files := make([]uploadFile, 0, len(pages))
	previous := 0
	for _, p := range pages {
		if p.Page <= previous || len(p.Data) == 0 {
			return "", fmt.Errorf("上传页面无效，请重新排版")
		}
		previous = p.Page
		files = append(files, uploadFile{Name: fmt.Sprintf("%s-page%d.gcode", prefix, p.Page), Page: p.Page, Bytes: append([]byte(nil), p.Data...)})
	}
	if len(files) == 0 {
		return "", fmt.Errorf("文档没有可书写笔画")
	}
	var idBytes [8]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(idBytes[:])
	m.mu.Lock()
	if err := m.readyLocked(); err != nil {
		m.mu.Unlock()
		return "", err
	}
	m.state.Phase = "uploading"
	m.state.JobID = id
	m.state.DocumentID = docID
	m.state.Page = files[0].Page
	m.state.TotalPages = len(files)
	m.state.PageOrdinal = 1
	m.state.UploadedPages, m.state.Progress = 0, 0
	m.setMessageLocked("printer.uploadingPages", files[0].Page, len(files))
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelUpload = cancel
	m.mu.Unlock()
	m.notify()
	go m.uploadPages(ctx, cancel, id, files)
	return id, nil
}

func (m *Manager) uploadPages(ctx context.Context, cancel context.CancelFunc, id string, files []uploadFile) {
	defer cancel()
	var total, sent int64
	for _, j := range files {
		total += int64(len(j.Bytes))
	}
	for i, j := range files {
		m.mu.Lock()
		if ctx.Err() != nil || m.state.Phase != "uploading" || m.state.JobID != id {
			m.mu.Unlock()
			return
		}
		m.state.Page, m.state.PageOrdinal = j.Page, i+1
		m.setMessageLocked("printer.uploadingPages", j.Page, len(files))
		m.mu.Unlock()
		m.notify()
		pageCtx, pageCancel := context.WithTimeout(ctx, 10*time.Minute)
		err := m.link.Upload(pageCtx, j.Name, j.Bytes, func(progress int) {
			m.mu.Lock()
			if m.state.Phase == "uploading" && m.state.JobID == id {
				m.state.Progress = min(99, int((sent*100+int64(len(j.Bytes))*int64(max(0, min(100, progress))))/total))
			}
			m.mu.Unlock()
			m.notify()
		})
		pageCancel()
		m.mu.Lock()
		if ctx.Err() != nil || m.state.Phase != "uploading" || m.state.JobID != id {
			m.mu.Unlock()
			return
		}
		if err != nil {
			m.state.Phase = "failed"
			m.setMessageLocked("printer.uploadPageFailed", j.Page, err)
			m.state.JobID = ""
			m.cancelUpload = nil
			m.mu.Unlock()
			m.notify()
			return
		}
		sent += int64(len(j.Bytes))
		m.state.UploadedPages = i + 1
		if i == len(files)-1 {
			m.state.Phase = "uploaded"
			m.state.Progress, m.state.JobID = 100, ""
			m.setMessageLocked("printer.uploadedPages", len(files))
			m.cancelUpload = nil
		}
		m.mu.Unlock()
		m.notify()
	}
}

// CancelUpload cancels the file transfer without sending any motion command.
func (m *Manager) CancelUpload() error {
	m.mu.Lock()
	if m.state.Phase != "uploading" {
		m.mu.Unlock()
		return fmt.Errorf("当前没有正在上传的文件")
	}
	if m.cancelUpload != nil {
		m.cancelUpload()
	}
	m.cancelUpload = nil
	m.state.Phase, m.state.JobID = "idle", ""
	m.setMessageLocked("printer.uploadCancelled")
	m.mu.Unlock()
	m.notify()
	return nil
}
