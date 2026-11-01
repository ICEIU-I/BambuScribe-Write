---
name: bambuscribe-table-import
description: 将表格截图导入 BambuScribe 当前客户端或 .ccf 项目，保留合并单元格、公式上下标及空白测量格，并自动摆放为可编辑表格。适用于图片转表格和连续追加实验表。
---

# 表格图片导入

使用项目 CLI 和本地排版器，将图片识别结果插入原生表格。Codex 自己读图即可，不要求额外模型 Key；客户端内置识别器是另一种入口。

## 识别规则

先查看用户的图片。读取 [识别规则](../../../desktop/internal/tableimport/instructions.txt) 和 [JSON Schema](../../../desktop/internal/tableimport/schema.json)，将结果保存为临时 extraction.json。同一张图中的表格按阅读顺序返回。

- 行列从 0 开始，合并单元格只写左上格；拓扑须完整且不重叠。
- 保留原图的表名、单位、合并和边线；表名及同一行参数合写到 title，不拆成 notes。测量格留空，省略号不补，不推算数据。
- 下标写作 `_{AK}` / `_{i}` / `_{s}`，上标用 Unicode。没有显式多行内容就不要添加换行。
- 字母上横线用紧随字母的 U+0305，例如 `U̅_{H}/mV`，填写横线用连续下划线；无横线的填写区保留连续空格，如 `d̅_{1}=        mm`。根号用 `√(d̅_{1}d̅_{2})` 指定横线范围；逐字保留下标大小写和正负号，不输出 LaTeX 命令。
- 看不清的格子标记 `uncertain:true`，保留能确认的文字，不能猜测。用户本轮要求优先于默认规则。

## 调用

优先使用 `desktop/build/bin/bambuscribe` 或 PATH 中的 `bambuscribe`。若没有包含导入命令的新版，从项目的 desktop 目录运行 `go build -trimpath -o build/bin/bambuscribe ./cmd/bambuscribe`。详细参数见 [CLI](../../../desktop/docs/cli.md)。

**当前客户端**：

1. `bambuscribe live snapshot` 读取当前页、正文排版和对象位置。快照的对象 page 从 0 开始，placement.page 从 1 开始；快照默认省略图片原始数据，不含连接凭据。
2. 保存临时 import.json：`{"extraction":识别结果,"placement":{"page":当前页,"y":插入起点}}`。用户指定某表下方时，用该表 ID 的 `afterId`；当前页末尾用 `atEnd:true`。位置留空时由编辑器按当前选择/光标决定。
3. `bambuscribe live import --params-file import.json`。客户端读取最新文档并重新计算摆放，将整组对象作为一次撤销操作。现有空表不能当作新表替换，除非用户明确要求补这个表。
4. 查看返回的页码与 warnings，说明不确定字符或缺字；需要目视验证时读取 `live snapshot --full`，将返回的 `project` 保存为临时 project.json，再运行 `bambuscribe new --params-file project.json --output temp.ccf` 和 `preview --project temp.ccf --format svg --output temp.svg`。不要读写 settings.json。

`expectedRevision` 可用于要求只在指定快照未变化时插入；通常省略它，让程序按最新手动调整摆放。请求失败或超时后先读取快照，确认是否已经插入，不要自动重复导入。

**指定 .ccf 文件**：

`bambuscribe import-table --project source.ccf --output result.ccf --params-file import.json`

再用 `check` 和 `preview` 校验。未指定覆盖时另存结果；不要擅自替换用户正在编辑的草稿。当前客户端不在线时，可按用户指定的项目文件完成导入；没有目标文件则询问目标，不猜测最后保存的项目。

排版器按当前正文字号测字，表名、说明和表格字号不低于正文的 70%（编辑器绝对下限 1 mm）；宽度不足时先在纸内加宽，再保留下标、根号及填写区进行换行和增高。不改变正文和已有对象。避让正文与对象，当前页不够时添加下一页。不可拆分公式仍超宽或整页高度不足时返回原因，按用户意图调整纸张或内容。导入完成后简短报告表格数、页码、字号调整和需修正内容。导入不包含设备上传或运动操作。
