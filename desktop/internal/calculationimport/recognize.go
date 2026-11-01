package calculationimport

import (
	"bambuscribe/desktop/internal/calculation"
	"bambuscribe/desktop/internal/tableimport"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed schema.json
var schema json.RawMessage

//go:embed instructions.txt
var instructions string

type Sources struct {
	Data        []string `json:"data"`
	References  []string `json:"references"`
	Instruction string   `json:"instruction"`
}

func Recognize(ctx context.Context, c tableimport.Config, s Sources) (calculation.Plan, error) {
	var empty calculation.Plan
	if err := c.Validate(); err != nil {
		return empty, err
	}
	if c.APIKey == "" {
		return empty, fmt.Errorf("请先在模型连接中填写API Key")
	}
	if len(s.Data) < 1 || len(s.Data) > 10 || len(s.References) < 1 || len(s.References) > 10 {
		return empty, fmt.Errorf("原始数据和参考样例各需1–10张图片")
	}
	if len([]rune(s.Instruction)) > 2000 {
		return empty, fmt.Errorf("补充要求最多2,000字")
	}
	images := []tableimport.ModelImage{}
	size := 0
	for role, group := range [][]string{s.Data, s.References} {
		for i, data := range group {
			if err := tableimport.ValidateImage(data); err != nil {
				return empty, err
			}
			size += len(data)
			if size > 64<<20 {
				return empty, fmt.Errorf("图片合计超过48 MB，请压缩后重试")
			}
			label := "原始数据"
			if role == 1 {
				label = "参考样例"
			}
			images = append(images, tableimport.ModelImage{Data: data, Label: fmt.Sprintf("%s第%d张", label, i+1)})
		}
	}
	raw, err := tableimport.RequestStructured(ctx, c, images, instructions, "按顺序结合整批图片生成数据处理方案。用户补充要求："+s.Instruction, schema, "bambuscribe_calculations", 32000)
	if err != nil {
		return empty, err
	}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	if len(raw) > 256<<10 {
		return empty, fmt.Errorf("识别内容过多，请分批导入")
	}
	return calculation.Decode([]byte(raw))
}
