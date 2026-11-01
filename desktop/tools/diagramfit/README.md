# 直接矢量重建试验

使用已配置的表格模型连接，发送原图和两个局部放大图，单次请求生成坐标 JSON。`schema.py` 定义非可执行图形格式，`request.py` 调用接口，`render.py` 使用给定参数直接绘制 PNG/SVG；不调用 OCR 或局部图线拟合。

从 `desktop` 执行，Python 需要 Pillow：

```sh
python tools/diagramfit/request.py --image internal/engine/testdata/hall-switch.png --output /tmp/diagram.json
go run ./tools/diagramfit/glyphs /tmp/diagram.json > /tmp/diagram-glyphs.json
python tools/diagramfit/render.py /tmp/diagram.json --glyphs /tmp/diagram-glyphs.json
```

默认读取 macOS 应用目录下的 `table-agent.json`，其他平台用 `--config` 指定。每次执行 `request.py` 都会向该服务发送图片并产生一次模型调用；已有输出文件不会覆盖，没有自动重试。只输出模型名、耗时和用量，不输出 Key。

客户端实现位于 `internal/diagramagent`，同一提示词和 JSON 格式，局部放大图最长边限制为 2048 像素。通用几何校验/圆与虚线采样位于 `internal/diagram`，手写和导出渲染位于 `internal/engine/image_diagram.go`。
