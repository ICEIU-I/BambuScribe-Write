package printer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"net"
	"sync"
	"time"
)

type Link interface {
	Connect(context.Context, Config, func(map[string]any), func(error)) error
	Publish(map[string]any) error
	Upload(context.Context, string, []byte, func(int)) error
	Close()
}

type Network struct {
	mu     sync.RWMutex
	client mqtt.Client
	config Config
}

func printerTLS(c Config) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true, ServerName: c.IP, ClientSessionCache: tls.NewLRUClientSessionCache(16), VerifyConnection: func(cs tls.ConnectionState) error {
		// Bambu uses self-signed certificates. Bind that certificate to the selected SN.
		if len(cs.PeerCertificates) == 0 || serialFromCN(cs.PeerCertificates[0].Subject.CommonName) != c.Serial {
			return fmt.Errorf("打印机证书与序列号不一致")
		}
		return nil
	}}
}

func (n *Network) Connect(ctx context.Context, c Config, onReport func(map[string]any), onLost func(error)) error {
	if err := c.Validate(); err != nil {
		return err
	}
	opts := mqtt.NewClientOptions().AddBroker("ssl://" + net.JoinHostPort(c.IP, "8883")).SetUsername("bblp").SetPassword(c.AccessCode).SetClientID(fmt.Sprintf("scribe-%d", time.Now().UnixNano())).SetTLSConfig(printerTLS(c)).SetConnectTimeout(6 * time.Second).SetWriteTimeout(6 * time.Second).SetAutoReconnect(false).SetCleanSession(true)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, e error) { onLost(e) })
	client := mqtt.NewClient(opts)
	n.mu.Lock()
	n.config = c
	n.client = client
	n.mu.Unlock()
	if err := waitToken(ctx, client.Connect()); err != nil {
		client.Disconnect(100)
		return fmt.Errorf("MQTT 连接失败：%w", err)
	}
	token := client.Subscribe("device/"+c.Serial+"/report", 0, func(_ mqtt.Client, m mqtt.Message) {
		if len(m.Payload()) > 2<<20 {
			return
		}
		var payload map[string]json.RawMessage
		if json.Unmarshal(m.Payload(), &payload) != nil {
			return
		}
		var p map[string]any
		if json.Unmarshal(payload["print"], &p) == nil && p != nil {
			onReport(p)
		}
	})
	if err := waitToken(ctx, token); err != nil {
		client.Disconnect(100)
		return err
	}
	return nil
}

func waitToken(ctx context.Context, t mqtt.Token) error {
	select {
	case <-t.Done():
		return t.Error()
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(8 * time.Second):
		return fmt.Errorf("通信超时")
	}
}

func (n *Network) Publish(payload map[string]any) error {
	n.mu.RLock()
	client, config := n.client, n.config
	n.mu.RUnlock()
	if client == nil || !client.IsConnectionOpen() {
		return fmt.Errorf("打印机未连接")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return waitToken(context.Background(), client.Publish("device/"+config.Serial+"/request", 0, false, data))
}

func (n *Network) Close() {
	n.mu.RLock()
	client := n.client
	n.mu.RUnlock()
	if client != nil {
		client.Disconnect(100)
	}
}
