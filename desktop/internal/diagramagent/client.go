// Package diagramagent reconstructs an image with exactly one model request.
package diagramagent

import (
	"bambuscribe/desktop/internal/diagram"
	"bambuscribe/desktop/internal/tableimport"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
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
	prompt := imagePrompt(w, h, instruction)
	for _, profile := range typography {
		prompt += "\nApplication typography: " + profile
	}
	images := make([]tableimport.ModelImage, len(pictures))
	for i, picture := range pictures {
		images[i] = tableimport.ModelImage{Data: picture}
	}
	options := tableimport.RequestOptions{}
	if strings.HasPrefix(c.Model, "gpt-6") {
		options.ReasoningEffort = "high"
	}
	text, err := tableimport.RequestStructured(ctx, c, images, instructions, prompt, schema, "bambuscribe_diagram", 18000, options)
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
