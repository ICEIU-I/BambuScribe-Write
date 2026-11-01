# 科学示意图 OCR 本机测试

比较 Apple Vision、PP-OCRv5 server、PP-OCRv6 medium，以及 Pix2Text MFD/MFR 1.5。只读图片，不连接客户端或打印机。模型第一次运行时下载，之后在本机推理。

样本为用户提供的 1062×812 霍尔效应图；`fixtures/hall-labels.json` 是人工标出的 28 个区域。裁切测试跳过检测步骤，不能视为整图自动识别的准确率。公式模型跳过中文图名，共 27 个裁切区域。结果保留低置信度输出，便于检查误识别，不意味着这些结果可以自动覆盖原图。

## 安装

使用 Python 3.12 和两个独立环境。Pix2Text 的 CnSTD 依赖与 RapidOCR 3.10.0 有导入冲突，因此公式环境固定为 RapidOCR 3.9.2。

在 `desktop/tools/ocrbench` 目录执行：

```sh
uv venv --python 3.12 /tmp/bambuscribe-ocr-paddle
uv pip install --python /tmp/bambuscribe-ocr-paddle/bin/python -r requirements-paddle.txt
uv venv --python 3.12 /tmp/bambuscribe-ocr-formula
uv pip install --python /tmp/bambuscribe-ocr-formula/bin/python -r requirements-formula.txt
```

## 整图与裁切

```sh
/tmp/bambuscribe-ocr-paddle/bin/python benchmark.py vision --image fixtures/hall-effect.png --output /tmp/vision.json
/tmp/bambuscribe-ocr-paddle/bin/python benchmark.py ppocrv5 --image fixtures/hall-effect.png --output /tmp/ppocrv5.json
/tmp/bambuscribe-ocr-paddle/bin/python benchmark.py ppocrv6 --image fixtures/hall-effect.png --scale 2 --output /tmp/ppocrv6.json
/tmp/bambuscribe-ocr-formula/bin/python benchmark.py pix2text --image fixtures/hall-effect.png --output /tmp/pix2text.json
/tmp/bambuscribe-ocr-paddle/bin/python benchmark.py ppocrv6 --image fixtures/hall-effect.png --labels fixtures/hall-labels.json --output /tmp/ppocrv6-crops.json
/tmp/bambuscribe-ocr-formula/bin/python benchmark.py pix2text --image fixtures/hall-effect.png --labels fixtures/hall-labels.json --output /tmp/pix2text-crops.json
```

Vision 需要 macOS 的 Swift/Vision。其配置与桌面端一致，返回归一化坐标；其他模型返回原图像素坐标。`--scale 2` 的结果已换算回原图坐标。Pix2Text 的 `--formula-width` 默认 768，可以设为 1536 检查检测尺寸敏感性。

结果记录输入 SHA-256、包版本、配置、文字、分数、坐标与耗时。整图先运行一次初始化/预热，再记录一次热运行；首次调用可能包含权重下载，不可作为纯推理耗时。分数不能跨模型比较，也不能当作正确率。

Paddle 模型默认保存在 `~/.cache/bambuscribe/ocr-eval/models`，可通过 `--model-cache` 修改；Pix2Text 使用其默认 `~/.pix2text`，CnOCR/CnSTD 使用各自缓存目录。依赖、模型及运行产物均不需要加入 Git。

实测结论、原始结果和权重哈希见评估记录（本机评估不随源码发布）。
