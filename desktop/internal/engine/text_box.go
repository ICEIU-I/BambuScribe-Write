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

func (f *Font) objectText(t TableAsset, text string, size, width float64, settings LayoutSettings) tableText {
	if t.isTextBox() {
		settings = settings.normalized()
		return f.tableTextWithSpacing(text, size, width, &settings)
	}
	return f.tableText(text, size, width)
}

func (f *Font) textBoxFontSize(t TableAsset, c TableCell, settings LayoutSettings) float64 {
	settings = settings.normalized()
	gap := settings.LetterGap
	settings.LetterGap = 0
	unit := f.tableTextWithSpacing(c.Text, 1, 1e9, &settings)
	xs, ys := tableOffsets(t.ColumnWidths), tableOffsets(t.RowHeights)
	width := xs[c.Col+c.ColSpan] - xs[c.Col] - 2*t.Padding
	height := ys[c.Row+c.RowSpan] - ys[c.Row] - 2*t.Padding
	size := 40.0
	for i, line := range unit.lines {
		if unit.widths[i] > 0 {
			// Millimeter gaps stay fixed when the font grows or shrinks.
			size = min(size, (width-textLineGap(line, gap))/unit.widths[i])
		}
	}
	if unit.height > 0 {
		size = min(size, height/unit.height)
	}
	return max(1, size)
}
