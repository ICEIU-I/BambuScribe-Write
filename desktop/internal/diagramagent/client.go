// Package diagramagent reconstructs an image with exactly one model request.
package diagramagent

import (
	"bambuscribe/desktop/internal/diagram"
	"bambuscribe/desktop/internal/tableimport"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//go:embed instructions.txt
var instructions string

//go:embed schema.json
var schema json.RawMessage

func Recognize(ctx context.Context, c tableimport.Config, data, instruction string, typography ...string) (*diagram.Scene, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("请先在模型连接中填写 API Key")
	}
	if len([]rune(instruction)) > 2000 {
		return nil, fmt.Errorf("补充要求最多 2,000 字")
	}
	pictures, w, h, err := prepareImages(data)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	format := map[string]any{"name": "bambuscribe_diagram", "strict": true, "schema": schema}
	prompt := imagePrompt(w, h, instruction)
	for _, profile := range typography {
		prompt += "\nApplication typography: " + profile
	}
	var body map[string]any
	endpoint := "/responses"
	if c.Protocol == "responses" {
		content := []any{map[string]any{"type": "input_text", "text": prompt}}
		for _, p := range pictures {
			content = append(content, map[string]any{"type": "input_image", "image_url": p, "detail": "high"})
		}
		format["type"] = "json_schema"
		body = map[string]any{"model": c.Model, "store": false, "instructions": instructions, "input": []any{map[string]any{"role": "user", "content": content}}, "text": map[string]any{"format": format}, "max_output_tokens": 18000}
		if strings.HasPrefix(c.Model, "gpt-6") {
			body["reasoning"] = map[string]any{"effort": "high"}
		}
	} else {
		endpoint = "/chat/completions"
		content := []any{map[string]any{"type": "text", "text": prompt}}
		for _, p := range pictures {
			content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": p, "detail": "high"}})
		}
		body = map[string]any{"model": c.Model, "messages": []any{map[string]any{"role": "system", "content": instructions}, map[string]any{"role": "user", "content": content}}, "response_format": map[string]any{"type": "json_schema", "json_schema": format}, "max_completion_tokens": 18000}
		if strings.HasPrefix(c.Model, "gpt-6") {
			body["reasoning_effort"] = "high"
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("模型接口地址无效")
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("图像模型连接失败或超时，请检查连接后重试")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("图像模型返回 HTTP %d，请检查模型连接与额度后重试", res.StatusCode)
	}
	raw, err = io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
	if err != nil || len(raw) > 4<<20 {
		return nil, fmt.Errorf("图像响应读取失败或过大，请裁剪后重试")
	}
	text, err := responseText(raw, c.Protocol)
	if err != nil {
		return nil, err
	}
	scene, err := diagram.Decode([]byte(text))
	if err != nil {
		return nil, err
	}
	if scene.Width != w || scene.Height != h {
		return nil, fmt.Errorf("模型更改了原图坐标，请重试")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return scene, nil
}

func responseText(raw []byte, protocol string) (string, error) {
	var r struct {
		Status string `json:"status"`
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &r) != nil {
		return "", fmt.Errorf("模型响应格式无效，请检查协议")
	}
	text := ""
	if protocol == "responses" {
		if r.Status != "completed" {
			return "", fmt.Errorf("图像识别未完成，请裁剪后重试")
		}
		for _, o := range r.Output {
			for _, p := range o.Content {
				if p.Type == "refusal" {
					return "", fmt.Errorf("模型未处理此图片，请更换图片")
				}
				if p.Type == "output_text" {
					text += p.Text
				}
			}
		}
	} else {
		if len(r.Choices) != 1 || r.Choices[0].Finish != "stop" || r.Choices[0].Message.Refusal != "" {
			return "", fmt.Errorf("图像识别未完成，请重试")
		}
		text = r.Choices[0].Message.Content
	}
	if text == "" {
		return "", fmt.Errorf("模型未返回图像内容，请重试")
	}
	return text, nil
}
