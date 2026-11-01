package cli

type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      string `json:"params"`
}

func Catalog() []ToolSpec {
	return []ToolSpec{
		{"recognize-table", "识别图片并输出表格结构（独立命令）", "--image PNG/JPEG,--config model.json?,--instruction?,--output extraction.json?"},
		{"import-table", "按正文字号自动摆放表格并保存项目（独立命令）", "--project source.ccf,--output result.ccf,--params-file {extraction,placement}"},
		{"live.snapshot", "读取当前客户端文档（live snapshot）", "--full?,--session editor-session.json?"},
		{"live.import", "向当前客户端插入表格并支持一次撤销（live import）", "--params-file {extraction,placement?,expectedRevision?},--session?"},
		{"text.set", "替换正文，显式换行保留", "text"},
		{"layout.set", "更新排版参数，未指定项保持", "LayoutSettings 字段，尺寸 mm、间距 %"},
		{"writing.set", "更新书写参数", "offsetX,offsetY,penUp,penDown,originMode,sound"},
		{"table.insert", "插入网格、三线、横线或无边框表格", "id?,page?,x?,y?,rows?,cols?,width?,height?,style?,fontSize?,padding?,align?,verticalAlign?,text?"},
		{"table.update", "更新表格尺寸、样式和对齐", "id,TableAsset 字段；page 从 1 开始，kind 不可改变"},
		{"table.cells", "更新选区文字、对齐及共享边框", "id,selection?,text?,align?,verticalAlign?,borders?:all|outer|none|reset"},
		{"table.merge", "合并矩形选区", "id,selection:{r0,c0,r1,c1}"},
		{"table.split", "拆分选中的合并单元格", "id,selection:{r0,c0,r1,c1}"},
		{"table.row.insert", "插入行", "id,index,size? (mm)"},
		{"table.row.delete", "删除行", "id,index"},
		{"table.column.insert", "插入列", "id,index,size? (mm)"},
		{"table.column.delete", "删除列", "id,index"},
		{"table.paste", "粘贴二维数据或 Excel TSV，必要时扩展行列", "id,row?,col?,data|tsv|file"},
		{"textbox.insert", "在纸面任意位置插入独立文本框", "id?,page?,x?,y?,width?,height?,fontSize?,padding?,align?,verticalAlign?,text?；固定 1×1 无边框"},
		{"textbox.update", "更新文字、位置、尺寸、字号或对齐", "id,text?,page?,x?,y?,width?,height?,fontSize?,padding?,align?,verticalAlign?"},
		{"image.insert", "插入 PNG/JPEG 图片并生成线稿", "id?,file,page?,x?,y?,width?,height?,rotation?"},
		{"image.update", "移动、缩放、旋转或替换图片", "id,file?,page?,x?,y?,width?,height?,rotation?"},
		{"image.text.insert", "插入旧式图片内覆盖文本框", "id,regionId?,text,x,y,width,height（图片局部 mm）"},
		{"image.text.update", "调整人工图片内文本框", "id,regionId,text?,x?,y?,width?,height?"},
		{"image.text.delete", "删除人工图片内文本框", "id,regionId"},
		{"image.erase.rect", "矩形擦除图片笔画", "id,regionId?,x,y,width,height（图片局部 mm）"},
		{"image.erase.path", "沿圆形笔刷轨迹擦除图片", "id,regionId?,points:[{x,y},...],radius?（图片局部 mm）"},
		{"image.erase.delete", "移除一次人工擦除记录", "id,regionId"},
		{"object.move", "移动对象，可跨页", "id,page?,x?,y?"},
		{"object.resize", "调整对象整体尺寸，默认保持字号", "id,width,height,scaleText?"},
		{"object.copy", "完整复制对象并放到指定位置", "id,newId?,page?,x?,y?"},
		{"object.delete", "删除表格、文本框或图片", "id"},
		{"undo", "撤销本批处理中的前一项修改", "{}"},
		{"redo", "重做本批处理中撤销的修改", "{}"},
		{"device.discover", "检测所选 IP 的序列号（device discover）", "--ip IP"},
		{"device.connect", "连接并获取首个状态，结束自动断开（device connect）", "--config device.json"},
		{"device.status", "连接后查询设备状态（device status）", "--config device.json"},
		{"device.watch", "保持连接并输出状态 JSONL（device watch）", "--config device.json,--timeout"},
		{"device.upload", "整批预检后上传，Ctrl+C 取消（device upload）", "--project note.ccf,--config device.json,--params {prefix,pages?}；pages 从 1 开始，省略上传全部"},
	}
}
