package tableimport

import (
	"bytes"
	"context"
	"encoding/json"
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
		body = map[string]any{"model": c.Model, "stream": true, "store": false, "instructions": instructions, "input": []any{map[string]any{"role": "user", "content": content}}, "text": map[string]any{"format": format}, "max_output_tokens": maxTokens}
	} else {
		content = append(content, map[string]any{"type": "text", "text": prompt})
		for _, im := range images {
			if im.Label != "" {
				content = append(content, map[string]any{"type": "text", "text": im.Label})
			}
			content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": im.Data, "detail": "high"}})
		}
		body = map[string]any{"model": c.Model, "stream": true, "messages": []any{map[string]any{"role": "system", "content": instructions}, map[string]any{"role": "user", "content": content}}, "response_format": map[string]any{"type": "json_schema", "json_schema": format}, "max_completion_tokens": maxTokens}
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
	req.Header.Set("Accept", "text/event-stream, application/json")
	client := &http.Client{Timeout: 300 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return "", requestError(ctx, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", statusError(res.StatusCode)
	}
	reportProgress(ctx, "processing")
	return readModelResponse(ctx, res.Body, res.Header.Get("Content-Type"), c.Protocol)
}
