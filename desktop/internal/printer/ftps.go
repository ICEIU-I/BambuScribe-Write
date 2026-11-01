package printer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/jlaffaye/ftp"
	"io"
	"net"
	"strings"
	"time"
)

type progressReader struct {
	ctx         context.Context
	reader      io.Reader
	total, read int
	last        int
	progress    func(int)
}

func (r *progressReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, e := r.reader.Read(b)
	r.read += n
	p := min(99, r.read*100/r.total)
	if p != r.last {
		r.last = p
		r.progress(p)
	}
	return n, e
}

func (n *Network) Upload(ctx context.Context, name string, data []byte, progress func(int)) error {
	if len(data) == 0 || strings.ContainsAny(name, "/\\\r\n") {
		return fmt.Errorf("上传文件无效")
	}
	n.mu.RLock()
	config := n.config
	n.mu.RUnlock()
	cfg := printerTLS(config)
	dialSocket := func(network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 15 * time.Second}
		raw, e := d.DialContext(ctx, network, address)
		if e != nil {
			return nil, e
		}
		context.AfterFunc(ctx, func() { raw.Close() })
		return tls.Client(&idleConn{Conn: raw}, cfg), nil
	}
	dial := func() (*ftp.ServerConn, error) {
		return ftp.Dial(net.JoinHostPort(config.IP, "990"), ftp.DialWithTLS(cfg), ftp.DialWithDialFunc(dialSocket), ftp.DialWithTimeout(15*time.Second), ftp.DialWithShutTimeout(10*time.Second), ftp.DialWithContext(ctx))
	}
	conn, err := dial()
	if err != nil {
		return fmt.Errorf("FTPS 连接失败：%w", err)
	}
	defer conn.Quit()
	if err = conn.Login("bblp", config.AccessCode); err != nil {
		return fmt.Errorf("FTPS 登录失败：%w", err)
	}
	r := &progressReader{ctx: ctx, reader: bytes.NewReader(data), total: len(data), progress: progress, last: -1}
	err = conn.Stor("/"+name, r)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		if r.read != len(data) {
			return fmt.Errorf("上传未完成：%w", err)
		}
		// A1 may close after the final block without 226; verify remote size on a new connection.
		verify, e := dial()
		if e != nil {
			return fmt.Errorf("无法确认上传完成：%w", err)
		}
		defer verify.Quit()
		if e = verify.Login("bblp", config.AccessCode); e != nil {
			return e
		}
		size, e := verify.FileSize("/" + name)
		if e != nil || size != int64(len(data)) {
			return fmt.Errorf("无法确认远端文件完整性，请重新上传")
		}
	}
	progress(100)
	return nil
}

// Bound idle network I/O as well as dial time; cancellation closes each socket.
type idleConn struct{ net.Conn }

func (c *idleConn) Read(p []byte) (int, error) {
	c.Conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	return c.Conn.Read(p)
}
func (c *idleConn) Write(p []byte) (int, error) {
	c.Conn.SetWriteDeadline(time.Now().Add(20 * time.Second))
	return c.Conn.Write(p)
}
