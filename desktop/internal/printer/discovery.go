package printer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

func Discover(ctx context.Context, ip string) (string, error) {
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("请输入有效 IP")
	}
	d := tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, "8883"))
	if err == nil {
		state := conn.(*tls.Conn).ConnectionState()
		conn.Close()
		if len(state.PeerCertificates) > 0 {
			if sn := serialFromCN(state.PeerCertificates[0].Subject.CommonName); sn != "" {
				return sn, nil
			}
		}
	}
	// Only accept an SSDP reply from the requested printer, not another LAN device.
	udp, e := net.ListenUDP("udp4", nil)
	if e != nil {
		return "", fmt.Errorf("无法自动检测序列号，请检查 IP 和网络后重试")
	}
	defer udp.Close()
	udp.SetDeadline(time.Now().Add(2 * time.Second))
	request := "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1990\r\nMAN: \"ssdp:discover\"\r\nMX: 1\r\nST: urn:bambulab-com:device:3dprinter:1\r\n\r\n"
	udp.WriteToUDP([]byte(request), &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1990})
	buf := make([]byte, 4096)
	for ctx.Err() == nil {
		n, addr, e := udp.ReadFromUDP(buf)
		if e != nil {
			break
		}
		if !addr.IP.Equal(net.ParseIP(ip)) {
			continue
		}
		for _, line := range strings.Split(string(buf[:n]), "\n") {
			if strings.HasPrefix(strings.ToLower(line), "usn:") {
				if s := serialFromCN(strings.TrimSpace(line[4:])); s != "" {
					return s, nil
				}
			}
		}
	}
	return "", fmt.Errorf("无法自动检测序列号，请确认 IP、打印机电源及 LAN 模式后重试")
}
