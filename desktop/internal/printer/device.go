package printer

import (
	"fmt"
	"net"
	"strings"
)

// Device is a locally saved connection profile. Each ID has its own session.
type Device struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Config
}

func ValidateDevices(devices []Device, selected string) error {
	ids, ips := map[string]bool{}, map[string]bool{}
	for _, d := range devices {
		if d.ID == "" || ids[d.ID] {
			return fmt.Errorf("实例编号无效，请重新新增")
		}
		ids[d.ID] = true
		if strings.TrimSpace(d.Name) == "" {
			return fmt.Errorf("请输入实例名称")
		}
		if d.IP != "" {
			ip := net.ParseIP(d.IP)
			if ip == nil {
				return fmt.Errorf("请输入有效 IP")
			}
			if ips[ip.String()] {
				return fmt.Errorf("此 IP 已有实例，请编辑已有实例")
			}
			ips[ip.String()] = true
		}
		if strings.ContainsAny(d.AccessCode, "\r\n\x00") {
			return fmt.Errorf("请输入有效访问码")
		}
	}
	if selected != "" && !ids[selected] {
		return fmt.Errorf("实例已删除，请重新选择")
	}
	return nil
}
