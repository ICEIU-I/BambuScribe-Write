package tableimport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// requestTableText sends one bounded model request using the selected model,
// high image detail, strict output and redacted transport errors.
func requestTableText(ctx context.Context, c Config, data, instructions, prompt string, schema json.RawMessage) (string, error) {
	return RequestStructured(ctx, c, []ModelImage{{Data: data}}, instructions, prompt, schema, "bambuscribe_tables", 18000)
}

// ModelImage preserves source roles and ordering in one multimodal request.
type ModelImage struct {
	Data  string
	Label string
}

func RequestStructured(ctx context.Context, c Config, images []ModelImage, instructions, prompt string, schema json.RawMessage, name string, maxTokens int) (string, error) {
	format := map[string]any{"name": name, "strict": true, "schema": schema}
	var body any
	endpoint := "/chat/completions"
	content := []any{}
	if c.Protocol == "responses" {
		endpoint = "/responses"
		format["type"] = "json_schema"
		content = append(content, map[string]any{"type": "input_text", "text": prompt})
		for _, im := range images {
			if im.Label != "" {
				content = append(content, map[string]any{"type": "input_text", "text": im.Label})
			}
			content = append(content, map[string]any{"type": "input_image", "image_url": im.Data, "detail": "high"})
		}
		body = map[string]any{"model": c.Model, "store": false, "instructions": instructions, "input": []any{map[string]any{"role": "user", "content": content}}, "text": map[string]any{"format": format}, "max_output_tokens": maxTokens}
	} else {
		content = append(content, map[string]any{"type": "text", "text": prompt})
		for _, im := range images {
			if im.Label != "" {
				content = append(content, map[string]any{"type": "text", "text": im.Label})
			}
			content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": im.Data, "detail": "high"}})
		}
		body = map[string]any{"model": c.Model, "messages": []any{map[string]any{"role": "system", "content": instructions}, map[string]any{"role": "user", "content": content}}, "response_format": map[string]any{"type": "json_schema", "json_schema": format}, "max_completion_tokens": maxTokens}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 300 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("模型接口连接失败或超时，请检查接口地址后重试")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		switch res.StatusCode {
		case 401, 403:
			return "", fmt.Errorf("模型接口拒绝访问（%d），请检查 API Key", res.StatusCode)
		case 429:
			return "", fmt.Errorf("模型接口额度不足或请求过多，请稍后重试")
		}
		return "", fmt.Errorf("模型接口返回 HTTP %d，请检查模型名、协议及图片/结构化输出支持", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
	if err != nil || len(b) > 4<<20 {
		return "", fmt.Errorf("模型响应过大或读取失败，请裁剪图片后重试")
	}
	var response struct {
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
	if err = json.Unmarshal(b, &response); err != nil {
		return "", fmt.Errorf("模型响应格式无效，请检查接口协议")
	}
	text := ""
	if c.Protocol == "responses" {
		if response.Status != "completed" {
			return "", fmt.Errorf("图片识别未完成，请裁剪图片后重试")
		}
		for _, o := range response.Output {
			for _, v := range o.Content {
				if v.Type == "refusal" {
					return "", fmt.Errorf("模型未识别此图片，请检查图片后重试")
				}
				if v.Type == "output_text" {
					text += v.Text
				}
			}
		}
	} else {
		if len(response.Choices) != 1 || response.Choices[0].Finish != "stop" || response.Choices[0].Message.Refusal != "" {
			return "", fmt.Errorf("图片识别未完成，请裁剪图片后重试")
		}
		text = response.Choices[0].Message.Content
	}
	if text == "" {
		return "", fmt.Errorf("模型没有返回内容，请检查图片后重试")
	}
	return text, nil
}
