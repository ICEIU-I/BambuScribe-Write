package tableimport

import (
	"context"
	"fmt"
)

// Recognize sends each image once. Structural checks are local; layout and
// missing-glyph checks run when planning the editable preview.
func Recognize(ctx context.Context, c Config, data, instruction string) (Extraction, error) {
	var empty Extraction
	if err := c.Validate(); err != nil {
		return empty, err
	}
	if c.APIKey == "" {
		return empty, fmt.Errorf("请先在模型连接中填写 API Key")
	}
	if err := ValidateImage(data); err != nil {
		return empty, err
	}
	if len([]rune(instruction)) > 2000 {
		return empty, fmt.Errorf("补充要求最多 2,000 字")
	}
	prompt := "识别图片中的表格。"
	if instruction != "" {
		prompt += "\n用户补充要求：" + instruction
	}
	raw, err := requestTableText(ctx, c, data, Instructions, prompt, Schema)
	if err != nil {
		return empty, err
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if len(raw) > 256<<10 {
		return empty, fmt.Errorf("识别内容过多，请拆分图片后重试")
	}
	result, err := Decode([]byte(raw))
	if err != nil {
		return empty, fmt.Errorf("识别结果无效：%w；请重试或裁剪图片", err)
	}
	result.Review = nil
	return result, nil
}
