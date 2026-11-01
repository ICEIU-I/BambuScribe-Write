package engine

// Tables retain their editable topology; all distances are paper millimeters.
type TableBorders struct {
	Top    *bool `json:"top,omitempty"`
	Right  *bool `json:"right,omitempty"`
	Bottom *bool `json:"bottom,omitempty"`
	Left   *bool `json:"left,omitempty"`
}

type TableCell struct {
	Row           int           `json:"row"`
	Col           int           `json:"col"`
	RowSpan       int           `json:"rowSpan"`
	ColSpan       int           `json:"colSpan"`
	Text          string        `json:"text"`
	Align         string        `json:"align,omitempty"`
	VerticalAlign string        `json:"verticalAlign,omitempty"`
	Borders       *TableBorders `json:"borders,omitempty"`
}

type TableAsset struct {
	GroupID       string      `json:"groupId,omitempty"`
	ID            string      `json:"id"`
	Kind          string      `json:"kind,omitempty"`
	Page          int         `json:"page"`
	X             float64     `json:"x"`
	Y             float64     `json:"y"`
	ColumnWidths  []float64   `json:"columnWidths"`
	RowHeights    []float64   `json:"rowHeights"`
	Cells         []TableCell `json:"cells"`
	Style         string      `json:"style"`
	FontSize      float64     `json:"fontSize"`
	Padding       float64     `json:"padding"`
	Align         string      `json:"align"`
	VerticalAlign string      `json:"verticalAlign"`
}

type TableCellPlacement struct {
	TableCell
	X           float64             `json:"x"`
	Y           float64             `json:"y"`
	Width       float64             `json:"width"`
	Height      float64             `json:"height"`
	FontSize    float64             `json:"fontSize"`
	TextWidth   float64             `json:"textWidth"`
	TextHeight  float64             `json:"textHeight"`
	TextSpacing *TextSpacingMetrics `json:"textSpacing,omitempty"`
	Strokes     []Stroke            `json:"strokes"`
	Missing     []MissingGlyph      `json:"missing"`
}

type TablePlacement struct {
	ID           string               `json:"id"`
	Kind         string               `json:"kind,omitempty"`
	FontSize     float64              `json:"fontSize"`
	Padding      float64              `json:"padding"`
	X            float64              `json:"x"`
	Y            float64              `json:"y"`
	Width        float64              `json:"width"`
	Height       float64              `json:"height"`
	ColumnWidths []float64            `json:"columnWidths"`
	RowHeights   []float64            `json:"rowHeights"`
	Cells        []TableCellPlacement `json:"cells"`
	Strokes      []Stroke             `json:"strokes"`
	Issues       []string             `json:"issues"`
}
