package tableimport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Stream framing is larger than the final JSON (deltas, heartbeats and snapshot).
const maxStreamBytes = 32 << 20

func readModelResponse(ctx context.Context, body io.Reader, contentType, protocol string) (string, error) {
	reader := bufio.NewReader(body)
	isStream := strings.HasPrefix(strings.ToLower(contentType), "text/event-stream")
	// Some compatible gateways omit the content type. Detect standard SSE prefixes
	// while retaining JSON responses from gateways that ignore stream=true.
	if !isStream {
		prefix, _ := reader.Peek(5)
		isStream = strings.HasPrefix(string(prefix), "data:") || strings.HasPrefix(string(prefix), "event") || strings.HasPrefix(string(prefix), ":")
	}
	if isStream {
		return readModelStream(ctx, reader, protocol)
	}
	b, err := io.ReadAll(io.LimitReader(reader, maxResponseBytes+1))
	if err != nil {
		return "", requestError(ctx, err)
	}
	if len(b) > maxResponseBytes {
		return "", fmt.Errorf("模型响应过大，请分批导入")
	}
	return decodeModelResponse(b, protocol)
}

func readModelStream(ctx context.Context, reader io.Reader, protocol string) (string, error) {
	limited := &io.LimitedReader{R: reader, N: maxStreamBytes + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), maxResponseBytes)
	var data, text strings.Builder
	eventName := ""
	receiving := false
	progress := func() {
		if !receiving {
			reportProgress(ctx, "receiving")
			receiving = true
		}
	}
	consume := func() (string, bool, error) {
		raw := strings.TrimSpace(data.String())
		data.Reset()
		name := eventName
		eventName = ""
		if raw == "" {
			return "", false, nil
		}
		if raw == "[DONE]" {
			return "", false, incompleteResponse()
		}
		var event struct {
			Type     string          `json:"type"`
			Delta    string          `json:"delta"`
			Response json.RawMessage `json:"response"`
			Error    json.RawMessage `json:"error"`
			Choices  []struct {
				Index  int    `json:"index"`
				Finish string `json:"finish_reason"`
				Delta  struct {
					Content string `json:"content"`
					Refusal string `json:"refusal"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(raw), &event) != nil {
			return "", false, fmt.Errorf("模型流式响应格式无效，请检查接口协议")
		}
		if event.Type == "" {
			event.Type = name
		}
		if hasModelError(event.Error) || event.Type == "error" || event.Type == "response.failed" {
			return "", false, fmt.Errorf("模型服务处理失败，请稍后重试或更换接口")
		}
		if protocol == "responses" {
			switch event.Type {
			case "response.output_text.delta":
				if event.Delta != "" {
					progress()
				}
			case "response.completed":
				// The final snapshot is authoritative. Do not duplicate streamed deltas
				// or accept a truncated stream just because its partial JSON parses.
				result, err := decodeModelResponse(event.Response, protocol)
				return result, true, err
			case "response.incomplete", "response.refusal.delta", "response.refusal.done":
				return "", false, incompleteResponse()
			}
			return "", false, nil
		}
		if len(event.Choices) > 1 {
			return "", false, incompleteResponse()
		}
		for _, choice := range event.Choices {
			if choice.Index != 0 || choice.Delta.Refusal != "" {
				return "", false, incompleteResponse()
			}
			if text.Len()+len(choice.Delta.Content) > maxResponseBytes {
				return "", false, fmt.Errorf("模型响应过大，请分批导入")
			}
			if choice.Delta.Content != "" {
				progress()
				text.WriteString(choice.Delta.Content)
			}
			if choice.Finish != "" {
				if choice.Finish != "stop" {
					return "", false, incompleteResponse()
				}
				result, err := completedText(text.String())
				return result, true, err
			}
		}
		return "", false, nil
	}
	for scanner.Scan() {
		if ctx.Err() != nil {
			return "", requestError(ctx, ctx.Err())
		}
		line := scanner.Text()
		if line == "" {
			if result, done, err := consume(); done || err != nil {
				return result, err
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
		if strings.HasPrefix(line, "data:") {
			part := strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")
			if data.Len()+len(part)+1 > maxResponseBytes {
				return "", fmt.Errorf("模型响应过大，请分批导入")
			}
			data.WriteString(part)
			data.WriteByte('\n')
		}
	}
	if limited.N == 0 || scanner.Err() == bufio.ErrTooLong {
		return "", fmt.Errorf("模型响应过大，请分批导入")
	}
	if err := scanner.Err(); err != nil {
		return "", requestError(ctx, err)
	}
	if ctx.Err() != nil {
		return "", requestError(ctx, ctx.Err())
	}
	return "", fmt.Errorf("模型连接中断，未收到完整结果，请重试")
}
