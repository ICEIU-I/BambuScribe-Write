# 本机图片 OCR 试验

> 2026-10-09：直接向纸面粘贴图片已恢复本机/系统 OCR，沿用现有配置，无需大模型 API Key。[图像 Agent](diagram-agent.md)保留在“智能导入 → 导入图像”中，按需手动使用。

图片粘贴可选用 PP-OCRv6 medium（RapidOCR / ONNX Runtime）检测和识别文字，再把独立的小文字区域交给 Pix2Text MFR 识别公式。只在识别结果一致且达到阈值时替换为内置手写字；中文标签需整图与紧裁剪结果一致：普通中文分数均不低于 0.92，单个汉字不低于 0.98；允许开头的 `(a)`、`(b)` 等分图编号，正文含单位或公式的拉丁/希腊字母时继续保留原图。单独的“一、二、十、丨”可能是图线或电荷标记，不替换；公式分数不低于 0.8，且与分数不低于 0.8 的整图或裁剪文本一致。分数不代表正确率，两种模型也可能同时识别错。

公式支持简单上下标、字母、数字与部分符号，例如 `F_{E}`、`E_{H}`、`10^{-11}`。对于带等号和单条分数线的紧凑公式截图，会把完整公式交给本机 Pix2Text；公式分数至少 0.9、与已有等式识别一致且通过受限语法检查时，按分子、分母、分数线和上下标结构绘制手写笔画。支持清除模型输出中的字号及间距标记，避免正确公式因 `\tiny` 等格式指令被丢弃。根式、箭头及堆叠上下标仍不替换；不能渲染的内容保留原笔画。文字的连通区域先从图线分离，避免字母竖笔、标点和下标进入表格直线及曲线处理。

这是可关闭的本机试验，不替代“智能导入”可编辑表格的模型配置。未配置时仍使用系统 OCR。配置启用后，运行环境缺失、进程失败或超时会回退到系统识别，并在页面提示原因；无检测结果也尝试系统识别。

## 配置

在应用配置目录放置 `image-ocr.json`，macOS 默认是 `~/Library/Application Support/BambuScribeWrite`；设置了 `BAMBUSCRIBE_CONFIG_DIR` 则使用该目录。

```json
{
  "enabled": true,
  "python": "/absolute/path/paddle-venv/bin/python",
  "formulaPython": "/absolute/path/formula-venv/bin/python",
  "modelDir": "/absolute/path/paddle-models",
  "formulaModelDir": "/absolute/path/mfr-1.5-onnx"
}
```

Paddle 模型目录须有 `PP-OCRv6_det_medium.onnx`、`PP-OCRv6_rec_medium.onnx` 及 RapidOCR 所需的识别字典。公式目录使用完整 MFR ONNX 模型包（包括 encoder、decoder、配置及 tokenizer），不要只复制权重。依赖安装和复跑方法见 [OCR 工具说明](../tools/ocrbench/README.md)。两个 Python 环境分开安装，避免 Pix2Text 的 CnSTD 依赖与新版 RapidOCR 冲突。

运行期间只读取本机模型，禁用 Hugging Face 在线下载。图片不发送给外部服务。模型脚本随应用打包；模型权重与 Python 环境留在本机，不进入 `.ccf`。第一次识别会启动模型，本次 1062×812 样图约 10 秒；总超时为 75 秒，公式子进程为 45 秒。

识别结果保存到配置目录的 `image-ocr-cache`，最多 32 张，包含识别文字与区域、不包含源图。源图字节、配置、脚本、模型大小或修改时间改变时重新识别。可删除该目录释放缓存。

将 `enabled` 改成 `false` 后重新打开应用即可停用。项目中已经保存的识别区域仍保留，若要重新识别那张图片，重新粘贴原图。旧版本未修改的自动 OCR 区域会刷新，人工文本框、擦除和已移动的旧对象保留。
