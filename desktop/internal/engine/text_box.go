package engine

// Text boxes share the single-cell handwriting layout and export stream, while
// retaining their independent editor, geometry and identity in saved projects.
func (t TableAsset) isTextBox() bool { return t.Kind == "textbox" }
func (t TableAsset) label() string {
	if t.isTextBox() {
		return "文本框"
	}
	return "表格"
}
