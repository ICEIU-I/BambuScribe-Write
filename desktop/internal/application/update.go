package application

import (
	"bambuscribe/desktop/internal/buildinfo"
	"bambuscribe/desktop/internal/i18n"
	"bambuscribe/desktop/internal/updater"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type updateState struct {
	mu       sync.Mutex
	client   *updater.Client
	release  *updater.Release
	prepared *updater.Prepared
	cancel   context.CancelFunc
	busy     bool
}

func updateError(err error) error {
	key := "installFailed"
	for _, pair := range []struct {
		err error
		key string
	}{
		{context.Canceled, "cancelled"}, {updater.ErrNetwork, "networkFailed"},
		{context.DeadlineExceeded, "networkFailed"},
		{updater.ErrPackage, "invalidPackage"}, {updater.ErrChecksum, "invalidPackage"},
		{updater.ErrLocation, "installLocation"}, {updater.ErrUnsupported, "unsupported"},
		{updater.ErrBusy, "busy"},
	} {
		if errors.Is(err, pair.err) {
			key = pair.key
			break
		}
	}
	return fmt.Errorf("%s", i18n.T("update."+key))
}

func (a *App) beginUpdate() (context.Context, *updater.Client, error) {
	a.updates.mu.Lock()
	defer a.updates.mu.Unlock()
	if a.updates.busy {
		return nil, nil, updateError(updater.ErrBusy)
	}
	if a.updates.client == nil {
		a.updates.client = updater.NewClient()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	a.updates.busy, a.updates.cancel = true, cancel
	return ctx, a.updates.client, nil
}

func (a *App) endUpdate() {
	a.updates.mu.Lock()
	defer a.updates.mu.Unlock()
	if a.updates.cancel != nil {
		a.updates.cancel()
	}
	a.updates.busy, a.updates.cancel = false, nil
}

func (a *App) CheckUpdate() (updater.Info, error) {
	ctx, client, err := a.beginUpdate()
	if err != nil {
		return updater.Info{}, err
	}
	defer a.endUpdate()
	checkCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	info, release, err := client.Check(checkCtx, buildinfo.Version)
	if err != nil {
		return info, updateError(err)
	}
	a.updates.mu.Lock()
	if a.updates.prepared != nil {
		a.updates.prepared.Cleanup()
		a.updates.prepared = nil
	}
	a.updates.release = release
	a.updates.mu.Unlock()
	return info, nil
}

// The frontend flushes inline edits and autosave before this call, and locks
// editing until it finishes. Never trust download URLs supplied by the UI.
func (a *App) InstallUpdate() error {
	ctx, client, err := a.beginUpdate()
	if err != nil {
		return err
	}
	defer a.endUpdate()
	a.updates.mu.Lock()
	release := a.updates.release
	a.updates.mu.Unlock()
	if release == nil {
		return updateError(updater.ErrPackage)
	}
	for _, d := range a.Printers().Devices {
		if d.State.Phase == "uploading" {
			return updateError(updater.ErrBusy)
		}
	}
	a.mu.Lock()
	appCtx := a.ctx
	a.mu.Unlock()
	prepared, err := client.Prepare(ctx, *release, func(progress updater.Progress) {
		if appCtx != nil {
			runtime.EventsEmit(appCtx, "update:progress", progress)
		}
	})
	if err != nil {
		return updateError(err)
	}
	a.updates.mu.Lock()
	if a.updates.prepared != nil {
		a.updates.prepared.Cleanup()
	}
	a.updates.prepared = prepared
	a.updates.mu.Unlock()
	return nil
}

// A second save after the download captures edits completed by background
// imports. The frontend calls this only after that save succeeds.
func (a *App) RestartUpdate() error {
	ctx, _, err := a.beginUpdate()
	if err != nil {
		return err
	}
	defer a.endUpdate()
	a.updates.mu.Lock()
	prepared, release := a.updates.prepared, a.updates.release
	a.updates.mu.Unlock()
	if prepared == nil || release == nil {
		return updateError(updater.ErrPackage)
	}
	if err := prepared.Start(ctx, strings.TrimPrefix(release.Tag, "v")); err != nil {
		return updateError(err)
	}
	a.updates.mu.Lock()
	a.updates.prepared = nil // The helper now owns the verified bundle.
	a.updates.mu.Unlock()
	a.mu.Lock()
	appCtx := a.ctx
	a.mu.Unlock()
	if appCtx != nil {
		runtime.EventsEmit(appCtx, "update:progress", updater.Progress{Phase: "restarting", Percent: 100})
	}
	a.CompleteClose()
	return nil
}

func (a *App) CancelUpdate() {
	a.updates.mu.Lock()
	defer a.updates.mu.Unlock()
	if a.updates.cancel != nil {
		a.updates.cancel()
	}
	if !a.updates.busy && a.updates.prepared != nil {
		a.updates.prepared.Cleanup()
		a.updates.prepared = nil
	}
}
