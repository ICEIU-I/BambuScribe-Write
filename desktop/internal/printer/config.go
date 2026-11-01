package printer

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type Config struct {
	IP         string `json:"ip"`
	AccessCode string `json:"accessCode"`
	Serial     string `json:"serial"`
}

func (c Config) Validate() error {
	if net.ParseIP(c.IP) == nil {
		return fmt.Errorf("请输入打印机的局域网 IP")
	}
	if c.AccessCode == "" || strings.ContainsAny(c.AccessCode, "\r\n\x00") {
		return fmt.Errorf("请输入有效访问码")
	}
	if !serialPattern.MatchString(c.Serial) {
		return fmt.Errorf("打印机序列号无效，请检查 IP 后重新连接")
	}
	return nil
}

var serialPattern = regexp.MustCompile(`^[A-Z0-9]{8,32}$`)

func serialFromCN(cn string) string {
	s := strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(cn)), ".BAMBULAB.COM")
	if serialPattern.MatchString(s) {
		return s
	}
	return ""
}
