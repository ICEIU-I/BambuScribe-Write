package application

import (
	"bambuscribe/desktop/internal/printer"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

type PrinterDevice struct {
	printer.Device
	State printer.State `json:"state"`
}
type PrinterList struct {
	Devices          []PrinterDevice `json:"devices"`
	SelectedDeviceID string          `json:"selectedDeviceId"`
}

func (a *App) Printers() PrinterList {
	list := PrinterList{Devices: []PrinterDevice{}}
	if a.store == nil {
		return list
	}
	d := a.store.Snapshot()
	list.SelectedDeviceID = d.SelectedDeviceID
	for _, device := range d.Devices {
		list.Devices = append(list.Devices, PrinterDevice{Device: device, State: a.fleet.State(device.ID)})
	}
	return list
}

func (a *App) SavePrinterDevice(device printer.Device) (PrinterList, error) {
	a.devicesMu.Lock()
	defer a.devicesMu.Unlock()
	if a.store == nil {
		return PrinterList{}, fmt.Errorf("配置目录不可用")
	}
	d := a.store.Snapshot()
	device.IP, device.Name = strings.TrimSpace(device.IP), strings.TrimSpace(device.Name)
	changedConnection := false
	if device.ID == "" {
		var id [12]byte
		if _, err := rand.Read(id[:]); err != nil {
			return PrinterList{}, err
		}
		device.ID = hex.EncodeToString(id[:])
		if device.Name == "" {
			for i := 1; ; i++ {
				name, used := fmt.Sprintf("A1 %d", i), false
				for _, existing := range d.Devices {
					if existing.Name == name {
						used = true
					}
				}
				if !used {
					device.Name = name
					break
				}
			}
		}
		device.Serial = ""
		d.Devices = append(d.Devices, device)
		if d.SelectedDeviceID == "" {
			d.SelectedDeviceID = device.ID
		}
	} else {
		found := false
		for i, old := range d.Devices {
			if old.ID != device.ID {
				continue
			}
			found = true
			state := a.fleet.State(device.ID)
			if state.Phase == "uploading" || state.Phase == "connecting" {
				return PrinterList{}, fmt.Errorf("请等待此实例连接或上传完成后编辑")
			}
			changedConnection = old.IP != device.IP || old.AccessCode != device.AccessCode
			device.Serial = old.Serial
			if changedConnection {
				device.Serial = ""
			}
			if device == old {
				return a.Printers(), nil
			}
			d.Devices[i] = device
		}
		if !found {
			return PrinterList{}, fmt.Errorf("实例已删除，请重新选择")
		}
	}
	if err := a.store.SaveDevices(d.Devices, d.SelectedDeviceID); err != nil {
		return PrinterList{}, err
	}
	a.fleet.Add(device.ID)
	if changedConnection {
		_ = a.fleet.Disconnect(device.ID)
	}
	return a.Printers(), nil
}

func (a *App) DeletePrinterDevice(id string) (PrinterList, error) {
	a.devicesMu.Lock()
	defer a.devicesMu.Unlock()
	if a.store == nil {
		return PrinterList{}, fmt.Errorf("配置目录不可用")
	}
	d := a.store.Snapshot()
	state := a.fleet.State(id)
	if state.Phase == "uploading" || state.Phase == "connecting" {
		return PrinterList{}, fmt.Errorf("请等待连接完成或取消上传后删除此实例")
	}
	devices := make([]printer.Device, 0, len(d.Devices))
	found := false
	for _, device := range d.Devices {
		if device.ID == id {
			found = true
		} else {
			devices = append(devices, device)
		}
	}
	if !found {
		return PrinterList{}, fmt.Errorf("实例已删除，请重新选择")
	}
	if d.SelectedDeviceID == id {
		d.SelectedDeviceID = ""
		if len(devices) > 0 {
			d.SelectedDeviceID = devices[0].ID
		}
	}
	if err := a.store.SaveDevices(devices, d.SelectedDeviceID); err != nil {
		return PrinterList{}, err
	}
	a.fleet.Remove(id)
	return a.Printers(), nil
}

func (a *App) SelectPrinter(id string) (PrinterList, error) {
	a.devicesMu.Lock()
	defer a.devicesMu.Unlock()
	if a.store == nil {
		return PrinterList{}, fmt.Errorf("配置目录不可用")
	}
	d := a.store.Snapshot()
	if err := a.store.SaveDevices(d.Devices, id); err != nil {
		return PrinterList{}, err
	}
	return a.Printers(), nil
}
