package tableimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
)

const maxResponseBytes = 4 << 20

type modelResponse struct {
	Status string          `json:"status"`
	Error  json.RawMessage `json:"error"`
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

func decodeModelResponse(b []byte, protocol string) (string, error) {
	var response modelResponse
	if json.Unmarshal(b, &response) != nil {
		return "", fmt.Errorf("模型响应格式无效，请检查接口协议")
	}
	if hasModelError(response.Error) {
		return "", fmt.Errorf("模型服务处理失败，请稍后重试或更换接口")
	}
	text := ""
	if protocol == "responses" {
		if response.Status != "completed" {
			return "", incompleteResponse()
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
			return "", incompleteResponse()
		}
		text = response.Choices[0].Message.Content
	}
	return completedText(text)
}

func completedText(text string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("模型没有返回内容，请检查图片后重试")
	}
	return text, nil
}

func hasModelError(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

func incompleteResponse() error {
	return fmt.Errorf("模型未完整生成识别结果，请减少图片或计算步骤后重试")
}

func statusError(status int) error {
	switch status {
	case 401, 403:
		return fmt.Errorf("模型接口拒绝访问（%d），请检查 API Key", status)
	case 429:
		return fmt.Errorf("模型接口额度不足或请求过多，请稍后重试")
	case 408, 504, 524:
		return fmt.Errorf("模型服务响应超时（HTTP %d），请稍后重试；若反复出现，请减少图片或更换模型接口", status)
	}
	if status >= 500 {
		return fmt.Errorf("模型服务暂时不可用（HTTP %d），请稍后重试或更换接口", status)
	}
	return fmt.Errorf("模型接口返回 HTTP %d，请检查模型名、协议及图片/结构化输出支持", status)
}

// Never expose provider bodies or transport URLs, which may contain secrets.
func requestError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	var timeout net.Error
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout() {
		return fmt.Errorf("模型处理超时，请减少图片或计算步骤后重试；也可更换模型接口：%w", context.DeadlineExceeded)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("模型连接中断，未收到完整结果，请重试")
	}
	return fmt.Errorf("模型接口连接失败，请检查网络和接口地址后重试")
}
