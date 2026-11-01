package tableimport

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"strings"
)

//go:embed schema.json
var Schema json.RawMessage

//go:embed instructions.txt
var Instructions string

type Config struct {
	BaseURL  string `json:"baseURL"`
	Model    string `json:"model"`
	Protocol string `json:"protocol"`
	APIKey   string `json:"apiKey,omitempty"`
	HasKey   bool   `json:"hasKey,omitempty"`
}

func (c Config) Validate() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("接口地址无效，请填写以 http:// 或 https:// 开头的 API 地址")
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("请填写支持图片识别的模型名")
	}
	if c.Protocol != "responses" && c.Protocol != "chat-completions" {
		return fmt.Errorf("请选择 Responses 或 Chat Completions 协议")
	}
	return nil
}
func ValidateImage(data string) error {
	if len(data) > 16<<20 {
		return fmt.Errorf("图片超过 12 MB，请裁剪后重试")
	}
	prefix, encoded, ok := strings.Cut(data, ",")
	if !ok || (prefix != "data:image/png;base64" && prefix != "data:image/jpeg;base64") {
		return fmt.Errorf("请选择 PNG 或 JPEG 图片")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) > 12<<20 {
		return fmt.Errorf("图片数据无效或超过 12 MB")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		return fmt.Errorf("图片无法读取或像素过多，请裁剪后重试")
	}
	return nil
}
