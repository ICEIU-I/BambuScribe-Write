package application

import (
	"bambuscribe/desktop/internal/printer"
	"context"
	"fmt"
	"strings"
	"time"
)

func (a *App) DiscoverPrinter(ip string) (string, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 8*time.Second)
	defer cancel()
	return printer.Discover(ctx, strings.TrimSpace(ip))
}

func (a *App) ConnectDevice(id string) error {
	for _, d := range a.Printers().Devices {
		if d.ID == id {
			return a.fleet.Connect(a.ctx, id, d.Config)
		}
	}
	return fmt.Errorf("实例已删除，请重新选择")
}
func (a *App) DisconnectDevice(id string) error   { return a.fleet.Disconnect(id) }
func (a *App) CancelDeviceUpload(id string) error { return a.fleet.CancelUpload(id) }

// Selected-device entry points preserve existing desktop integrations.
func (a *App) ConnectPrinter(c printer.Config) error {
	list := a.Printers()
	d := printer.Device{Config: c}
	for _, item := range list.Devices {
		if item.ID == list.SelectedDeviceID {
			d.ID, d.Name = item.ID, item.Name
			if c.AccessCode == "" && c.IP == item.IP {
				d.AccessCode = item.AccessCode
			}
		}
	}
	next, err := a.SavePrinterDevice(d)
	if err != nil {
		return err
	}
	return a.ConnectDevice(next.SelectedDeviceID)
}
func (a *App) DisconnectPrinter() error    { return a.DisconnectDevice(a.Printers().SelectedDeviceID) }
func (a *App) PrinterState() printer.State { return a.fleet.State(a.Printers().SelectedDeviceID) }
func (a *App) CancelUpload() error         { return a.CancelDeviceUpload(a.Printers().SelectedDeviceID) }
