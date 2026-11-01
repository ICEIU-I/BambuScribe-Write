package storage

import "bambuscribe/desktop/internal/printer"

func normalizeDevices(data *Data) {
	if data.Devices == nil {
		data.Devices = []printer.Device{}
		if data.Printer.IP != "" || data.Printer.AccessCode != "" {
			data.Devices = append(data.Devices, printer.Device{ID: "primary", Name: "A1 1", Config: data.Printer})
			data.SelectedDeviceID = "primary"
		}
	}
	if data.SelectedDeviceID == "" && len(data.Devices) > 0 {
		data.SelectedDeviceID = data.Devices[0].ID
	}
}

func (s *Store) SaveDevices(devices []printer.Device, selected string) error {
	if err := printer.ValidateDevices(devices, selected); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Devices = append([]printer.Device{}, devices...)
	next.SelectedDeviceID = selected
	next.Printer = printer.Config{}
	for _, d := range devices {
		if d.ID == selected {
			next.Printer = d.Config
		}
	}
	return s.save(next)
}
