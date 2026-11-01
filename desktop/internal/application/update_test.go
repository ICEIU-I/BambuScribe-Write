package application

import (
	"bambuscribe/desktop/internal/updater"
	"context"
	"errors"
	"testing"
)

func TestUpdateOperationsSerializeAndCanBeCancelled(t *testing.T) {
	a := &App{}
	ctx, _, err := a.beginUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.beginUpdate(); err == nil {
		t.Fatal("overlapping update accepted")
	}
	a.CancelUpdate()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("cancel did not reach downloader")
	}
	a.endUpdate()
	if _, _, err := a.beginUpdate(); err != nil {
		t.Fatal("could not retry after cancellation")
	}
	a.endUpdate()
	if err := a.InstallUpdate(); err == nil {
		t.Fatal("install accepted without a checked release")
	}
	if updateError(updater.ErrChecksum).Error() == "" {
		t.Fatal("empty error message")
	}
}
