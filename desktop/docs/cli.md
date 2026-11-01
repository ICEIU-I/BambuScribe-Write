# 命令行工具

`bambuscribe` 可以独立运行，不需要启动客户端。所有编辑工具读取指定的 `.ccf` 项目并原子保存到 `--output`；在客户端打开结果即可继续编辑。预览、校验、导出和上传共用客户端的内置字库、图片处理、坐标校验及 G-code 引擎。

## 构建和调用

```sh
cd desktop
go build -trimpath -o build/bin/bambuscribe ./cmd/bambuscribe
./build/bin/bambuscribe help
./build/bin/bambuscribe tools
```

`tools` 返回完整工具目录及参数。普通命令的结果是 stdout JSON，错误是 stderr JSON，错误退出码为 1；Ctrl+C 取消时为 130。设备上传进度写到 stderr JSONL，`device watch` 的状态流写到 stdout JSONL。`preview --output -` 直接输出 SVG／预览 JSON，适合重定向。

编辑参数使用 `--params 'JSON'` 或 `--params-file 文件`，二者选一；`--params-file -`、`--ops-file -` 表示从 stdin 读取。内容含换行、引号或复杂公式时，建议使用文件或 stdin，避免 shell 转义改变文字。文件参数中的相对路径按当前工作目录解析。

纸面位置、尺寸、字号、内边距和书写坐标的单位都是 mm，旋转单位为度；`letterSpacing` 和 `lineSpacing` 为百分比。工具参数 `page`、导出 `--page` 和上传 `pages` 从 **1** 开始。单元格行列和插入／删除索引从 **0** 开始，选区终点包含在内。`inspect` 返回项目存储模型，其中图片／表格的 `page` 从 **0** 开始；预览中的 `pages[].number` 从 1 开始。

## 项目、排版和导出

```sh
bambuscribe new --output note.ccf
bambuscribe inspect --project note.ccf
bambuscribe tool text.set --project note.ccf --output note.ccf --params '{"text":"实验一\n实验内容"}'
bambuscribe tool layout.set --project note.ccf --output note.ccf --params '{"fontSize":8,"letterSpacing":85,"lineSpacing":100}'
bambuscribe tool layout.set --project note.ccf --output note.ccf --params '{"fontId":"dc"}'
bambuscribe tool writing.set --project note.ccf --output note.ccf --params '{"penUp":7,"penDown":3.4,"originMode":"auto","sound":true}'
bambuscribe save --project note.ccf --output copy.ccf
bambuscribe preview --project note.ccf --format svg --page 1 --output page.svg
bambuscribe preview --project note.ccf --format json --output preview.json
bambuscribe check --project note.ccf
bambuscribe export --project note.ccf --page 1 --output page.gcode
bambuscribe export --project note.ccf --page all --output pages.zip
```

`new --params` 可指定 `text`、`layout`、`writing` 等项目字段，未指定项使用默认值。`layout.set` 使用 `LayoutSettings` 字段，`fontSize` 是保存格式的毫米字框大小（客户端显示百分比）；`writing.set` 使用 `offsetX`、`offsetY`、`penUp`、`penDown`、`originMode`（`auto`／`manual`）、`sound`。所有笔画统一使用基础抬落笔 Z；旧项目中的底部降笔参数会忽略并在重新保存时移除。示例抬落笔值是默认值，实际使用需按笔架校准。

`inspect` 默认省略图片的 base64 原始数据，`--full` 保留完整项目数据。保存时只存源数据与可编辑对象，预览结果不写回项目。宽度不足、缺字或表格／文本框越界可先保存继续修改，`check` 会逐页返回问题，导出与上传会阻止输出这些页。结构错误、无效参数或超出纸张的图片会拒绝保存。空页不生成书写文件，全部导出和上传保留原页码。

## 表格与文本框

```sh
bambuscribe tool table.insert --project note.ccf --output note.ccf --params '{"id":"table1","page":1,"x":15,"y":50,"rows":3,"cols":3,"width":120,"style":"three-line","fontSize":4}'
bambuscribe tool table.paste --project note.ccf --output note.ccf --params '{"id":"table1","data":[["电压","电流","单位"],["1","2","A"]]}'
bambuscribe tool table.merge --project note.ccf --output note.ccf --params '{"id":"table1","selection":{"r0":0,"c0":0,"r1":0,"c1":1}}'
bambuscribe tool table.split --project note.ccf --output note.ccf --params '{"id":"table1","selection":{"r0":0,"c0":0,"r1":0,"c1":1}}'
bambuscribe tool table.cells --project note.ccf --output note.ccf --params '{"id":"table1","selection":{"r0":0,"c0":0,"r1":0,"c1":2},"align":"center","borders":"outer"}'
bambuscribe tool table.row.insert --project note.ccf --output note.ccf --params '{"id":"table1","index":1,"size":10}'
bambuscribe tool table.column.delete --project note.ccf --output note.ccf --params '{"id":"table1","index":2}'
bambuscribe tool table.update --project note.ccf --output note.ccf --params '{"id":"table1","columnWidths":[60,60],"style":"grid","padding":1}'
bambuscribe tool textbox.insert --project note.ccf --output note.ccf --params '{"id":"title","text":"表1 实验数据","page":1,"x":15,"y":30,"width":100,"fontSize":5}'
bambuscribe tool textbox.update --project note.ccf --output note.ccf --params '{"id":"title","text":"表1 电流关系","align":"center","width":110}'
```

表格样式为 `grid`、`three-line`、`horizontal` 或 `none`。`table.update` 可更新保存模型中的 `columnWidths`、`rowHeights`、`fontSize`、`padding`、`align`、`verticalAlign`、`style`、位置和 `cells`；直接提供 `cells` 时必须覆盖完整且不重叠的拓扑，日常增删行列请用专门工具。

`table.row.insert/delete`、`table.column.insert/delete` 使用 `index`，插入可指定 `size`。`table.cells` 可更新 `text`、`align`（`left/center/right`）、`verticalAlign`（`top/middle/bottom`）及共享边框（`all/outer/none/reset`），省略 `selection` 时作用于左上单元格；触及合并单元格的选区自动扩展。

`table.paste` 指定二维 `data`、`tsv` 字符串或 TSV `file` 三者之一，可指定起点 `row/col`，支持 Excel 的引号、单元格内换行和空列，并按需扩展表格。文本框固定为无边框的一个单元格，`textbox.update` 可修改 `text`、`page`、`x/y`、`width/height`、`fontSize`、`padding` 和对齐。

表格和独立文本框在每次排版时按每个非空单元格的实际笔画边界独立计算最大可用字号（1–40 mm），不受保存的旧 `fontSize` 限制；有效字号见预览的 `tables[].cells[].fontSize`；`tables[].fontSize` 为非空单元格的最小字号摘要。`textbox.insert` 的 `fontSize` 仅参与默认框高的初始计算，最终字号按框的宽高适配。

没有 `\n` 时，表格和文本框中的文字保持一行；JSON 字符串中的 `\n` 对应客户端的 Shift+Enter。字号按宽高、合并区域和显式换行自动放大或缩小；缩至 1 mm 仍超宽时阻止导出，高度不足时保留撑高保护。

表格和独立文本框支持 `_{内容}` 下标写法，例如 `"text":"U_{AK}/V"` 与 `"text":"表1 I-U_{AK}关系"`。预览与导出会等比缩小并下移原字形，编辑／保存保留标记；正文和旧式图片内文本框不解析此标记。

平均值上横线使用组合字符，例如 `"text":"U̅_{H}/mV"`（U 后是 U+0305）；它附着于前一个字母，不产生额外占位或缺字。连续下划线（如 `"text":"X_{c}=________ mm"`）绘制为一条连续填写横线。

## 图片、擦除和对象操作

```sh
bambuscribe tool image.insert --project note.ccf --output note.ccf --params '{"id":"figure","file":"diagram.png","page":1,"x":20,"y":80,"width":90}'
bambuscribe tool image.update --project note.ccf --output note.ccf --params '{"id":"figure","rotation":10}'
bambuscribe tool image.erase.rect --project note.ccf --output note.ccf --params '{"id":"figure","regionId":"erase1","x":10,"y":10,"width":5,"height":5}'
bambuscribe tool image.erase.path --project note.ccf --output note.ccf --params '{"id":"figure","regionId":"erase2","points":[{"x":20,"y":15},{"x":25,"y":20}],"radius":1}'
bambuscribe tool image.erase.delete --project note.ccf --output note.ccf --params '{"id":"figure","regionId":"erase1"}'
bambuscribe tool object.move --project note.ccf --output note.ccf --params '{"id":"figure","page":2,"x":20,"y":30}'
bambuscribe tool object.resize --project note.ccf --output note.ccf --params '{"id":"figure","width":100,"height":50}'
bambuscribe tool object.copy --project note.ccf --output note.ccf --params '{"id":"title","newId":"title2","page":2,"x":15,"y":15}'
bambuscribe tool object.delete --project note.ccf --output note.ccf --params '{"id":"title2"}'
```

图片未指定 `height` 时保持原图比例；`image.update` 可指定 `file` 替换原图，并保留人工文本框及擦除。擦除的矩形、轨迹点和半径使用**图片未旋转时左上角为原点的局部毫米坐标**，随图片缩放／旋转移动；圆形笔刷半径为图片短边的 0.2–15%，默认 1.8%。一次点击可传一个轨迹点。

旧式图片内覆盖文本框使用 `image.text.insert/update/delete`：参数为图片 `id`、区域 `regionId`、`text`、`x/y`、`width/height`，位置和尺寸同样使用图片局部 mm；它会覆盖底图。独立 `textbox.insert` 使用纸面坐标，允许与图片和表格重叠，正文自动避让。正文按整行避让图片、表格和独立文本框；对象进入正文行时，整行移到下方或下一页，不将正文拆到对象两侧或图间空隙。图片的实际笔画未占到正文行时保留原有位置，对象互相重叠时保持指定位置，仅自身超出纸张才整块换页；预览 JSON 的 `objectLayout` 返回实际位置及原始 `source` 坐标，`.ccf` 保留原始摆放位置。

通用 `object.move/resize/copy/delete` 适用于表格、独立文本框和整图。复制保留图片、擦除、文字和表格拓扑；省略 `newId` 自动生成 ID，返回 JSON 中可获取。表格和独立文本框随尺寸自动采用最大可用字号，无需传 `scaleText:true`。

## 批处理与撤销

```json
[
  {
    "tool": "textbox.insert",
    "params": { "id": "batch-title", "text": "实验数据", "x": 20, "y": 20 }
  },
  {
    "tool": "object.move",
    "params": { "id": "batch-title", "page": 2, "x": 20, "y": 15 }
  },
  { "tool": "undo" },
  { "tool": "redo" }
]
```

```sh
bambuscribe batch --project note.ccf --output result.ccf --ops-file operations.json
```

一次批处理最多 200 项编辑操作，全部成功后只保存一次；任何操作失败都保留原文件。`undo/redo` 针对同一批处理中前面的编辑，新的编辑清空重做分支。独立调用之间保留输入／输出文件作为版本；CLI 的批处理历史不写入 `.ccf`，客户端打开结果后使用自己的编辑历史。设备操作单独调用，不混入编辑事务。

## 设备通信

创建 `device.json`，访问码可省略并通过环境变量提供：

```json
{ "ip": "192.0.2.10", "serial": "实际打印机序列号" }
```

```sh
read -s BAMBUSCRIBE_ACCESS_CODE
export BAMBUSCRIBE_ACCESS_CODE
bambuscribe device discover --ip 192.0.2.10
bambuscribe device connect --config device.json
bambuscribe device status --config device.json
bambuscribe device watch --config device.json --timeout 10m
bambuscribe device upload --config device.json --project note.ccf --params '{"prefix":"31"}' --timeout 20m
bambuscribe device upload --config device.json --project note.ccf --params '{"prefix":"31","pages":[1,3]}'
unset BAMBUSCRIBE_ACCESS_CODE
```

配置也支持 `accessCode`，环境变量有值时优先使用；不会写入桌面配置或结果。省略序列号时自动检测，也可用 `--serial` 覆盖。发现／连接／查询默认超时 30 秒，上传默认 10 分钟；`--timeout` 可调整。`connect/status` 等待真实设备状态后返回并关闭连接；`watch` 持续输出状态，Ctrl+C 结束。每次调用管理自己的连接，不影响客户端的连接。

上传前先验证整批文件及设备空闲状态，文件名为 `<prefix>-page<N>.gcode`；省略 `pages` 上传全部，指定页码须递增且不重复。失败、断线或打印机进入运行状态会中止后续页面；Ctrl+C 取消传输并关闭连接。全部校验在网络连接前完成，选择单页时仅校验该页。与客户端一样，CLI 只上传文件，归零、启动、暂停和停止书写均在 A1 上操作。

本版的设备连接、上传、进度、取消及失败中止已使用模拟连接验证；真实 A1 上传仍需实机复核。

## 图片识别与自动导入

客户端工具栏的“智能导入”支持选择、拖入、粘贴 PNG/JPEG，支持多张图片队列，最多 3 张并行识别，每张调用一次模型，完成后在预览中修改和拖动摆放，确认后生成可编辑表格、表名与说明；优先使用当前正文字号，不自动折行、不填测量格和省略号。宽度不足时，表名和各条说明分别缩小字号，表格先确定初始行高，再按单元格独立适配实际笔画字号，并提示调整后的毫米值；不改变正文和已有对象。最小字号为 1 mm，仍超宽或超过整页可用高度时拒绝插入并说明原因。插入位置可选光标附近、当前页末尾或指定表格下方；避开正文、图片及已有表格，空间不足时顺延下一页。多个表格一起插入、一起撤销。

模型连接支持 Responses 与 Chat Completions 图片输入及严格 JSON Schema。接口地址、模型和 Key 在导入窗口中配置，另存本机 `BambuScribeWrite/table-agent.json`（0600），不进入项目、Git 或文档快照；界面只返回是否已保存 Key。图片只在主动点击“识别并预览”时发送到所选接口，支持取消。

```sh
bambuscribe recognize-table --image table.png --output extraction.json
bambuscribe recognize-table --image table.png --config model.json --instruction '测量格留空' --output extraction.json
bambuscribe live snapshot
bambuscribe live import --params-file import.json
bambuscribe import-table --project note.ccf --output edited.ccf --params-file import.json
```

`recognize-table` 默认使用客户端的模型连接，可用 `--config` 指定模型配置文件、`BAMBUSCRIBE_AGENT_KEY` 覆盖 Key，`--timeout` 默认 3 分钟。每张图片只调用一次模型，结构校验在本地完成；排版预览保留缺字和不确定提示。结果为 `{tables:[...]}`，完整结构见 [schema](../internal/tableimport/schema.json) 和 [识别规则](../internal/tableimport/instructions.txt)。也可以由 Codex skill 直接读图生成相同数据，无需调用第二个模型。

`import.json` 包装识别结果与摆放方式：

```json
{
  "extraction": {
    "tables": [{
      "title": "表1 实验数据", "notes": [], "rows": 2,
      "columns": [1, 1], "style": "grid",
      "cells": [
        {"row":0,"col":0,"rowSpan":1,"colSpan":1,"text":"U_{AK}/V","uncertain":false,"borders":null},
        {"row":0,"col":1,"rowSpan":1,"colSpan":1,"text":"I/A","uncertain":false,"borders":null},
        {"row":1,"col":0,"rowSpan":1,"colSpan":1,"text":"","uncertain":false,"borders":null},
        {"row":1,"col":1,"rowSpan":1,"colSpan":1,"text":"","uncertain":false,"borders":null}
      ]
    }]
  },
  "placement": {"page": 1, "y": 40}
}
```

`columns` 是正数列宽比例；`row/col` 从0开始，`rowSpan/colSpan` 从1开始。`borders:null` 继承表格样式，也可指定 top/right/bottom/left 的布尔值（null 继承）。`placement.page` 从1开始，`y` 单位 mm，`afterId` 指定现有表格或文本框，`atEnd:true` 接到当前页内容下面。`notes` 作为表名前后说明中的表前文字，保持阅读顺序。

`live` 通过客户端启动时创建的本机认证会话工作，不修改磁盘草稿。`snapshot` 返回 `{project, revision, page, projectPath}`，默认省略图片原数据，`--full` 保留；`--session` 可指定 `editor-session.json`，默认应用配置目录。`live import` 可省略 placement 使用当前选择/光标，可指定 `expectedRevision` 在文档已变化时拒绝执行；不指定时始终基于最新文档规划。返回对象、页码和识别/缺字提示。超时后先检查快照，确认未插入再重试，避免重复添加。

项目专用 Codex skill 位于 `.agents/skills/bambuscribe-table-import`，在本仓库可使用 `$bambuscribe-table-import`，无需安装全局 skill。

导入参数可增加 `placement.positions`，按识别结果中的表格顺序提供完整的 `{page,x,y,width}` 数组，页码从 1 开始、其余单位为 mm。每项定位表格连同表名和说明；明确位置允许与已有对象及其他导入组重叠，正文自动整行避让，越界仍会阻止插入。返回 plan 含 groups（组边界和对象 ID）与 preview（与最终导出共用的手写笔画）。桌面拖动使用保存的 groupId 同步移动表名、说明；CLI 的单对象编辑仍按指定 ID 操作。
