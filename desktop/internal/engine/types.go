package engine

import "bambuscribe/desktop/internal/diagram"

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Stroke []Point

type RegionBounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type ImageTextRegion struct {
	ID            string        `json:"id"`
	Text          string        `json:"text"`
	X             float64       `json:"x"`
	Y             float64       `json:"y"`
	Width         float64       `json:"width"`
	Height        float64       `json:"height"`
	Mode          string        `json:"mode"`
	Source        string        `json:"source,omitempty"`
	Confidence    float64       `json:"confidence,omitempty"`
	TextFormat    string        `json:"textFormat,omitempty"`
	OCRVersion    int           `json:"ocrVersion,omitempty"`
	SourceBounds  *RegionBounds `json:"sourceBounds,omitempty"`
	SourceStrokes []Stroke      `json:"sourceStrokes"`
	Strokes       []Stroke      `json:"strokes,omitempty"`
	ErasePath     []Point       `json:"erasePath,omitempty"`
	EraseRadius   float64       `json:"eraseRadius,omitempty"`
}

// ImageAsset keeps the editable raster source and its placement. The source
// is retained so resizing can regenerate clean strokes for the final G-code.
type ImageAsset struct {
	Diagram     *diagram.Scene    `json:"diagram,omitempty"`
	ID          string            `json:"id"`
	Page        int               `json:"page"`
	Data        string            `json:"data"`
	X           float64           `json:"x"`
	Y           float64           `json:"y"`
	Width       float64           `json:"width"`
	Height      float64           `json:"height"`
	Rotation    float64           `json:"rotation"`
	Mode        string            `json:"mode"`
	TextRegions []ImageTextRegion `json:"textRegions"`
}

type ImagePlacement struct {
	ID             string            `json:"id"`
	X              float64           `json:"x"`
	Y              float64           `json:"y"`
	Width          float64           `json:"width"`
	Height         float64           `json:"height"`
	Rotation       float64           `json:"rotation"`
	Mode           string            `json:"mode"`
	SourceStrokes  []Stroke          `json:"sourceStrokes"`
	GraphicStrokes []Stroke          `json:"graphicStrokes"`
	Strokes        []Stroke          `json:"strokes,omitempty"`
	TextRegions    []ImageTextRegion `json:"textRegions"`
}

type LayoutSettings struct {
	HandwritingVariation float64      `json:"handwritingVariation,omitempty"`
	HandwritingSeed      uint32       `json:"handwritingSeed,omitempty"`
	FractionScale        float64      `json:"fractionScale,omitempty"`
	FontID               string       `json:"fontId,omitempty"`
	PageLayouts          []PageLayout `json:"pageLayouts,omitempty"`
	TemplateName         string       `json:"templateName"`
	LineMode             string       `json:"lineMode"`
	LineCount            int          `json:"lineCount"`
	PaperWidth           float64      `json:"paperWidth"`
	PaperHeight          float64      `json:"paperHeight"`
	MarginTop            float64      `json:"marginTop"`
	MarginRight          float64      `json:"marginRight"`
	MarginBottom         float64      `json:"marginBottom"`
	MarginLeft           float64      `json:"marginLeft"`
	FontSize             float64      `json:"fontSize"`
	LetterGap            float64      `json:"letterGap"`
	LetterSpacing        float64      `json:"letterSpacing"`
	LineSpacing          float64      `json:"lineSpacing"`
	LineHeight           float64      `json:"lineHeight"`
	ParagraphGap         float64      `json:"paragraphGap"`
	Indent               float64      `json:"indent"`
	Align                string       `json:"align"`
}

// Explicit sheets retain their grid and paragraph continuation when copied.
type PageLayout struct {
	LineCount    int  `json:"lineCount"`
	Continuation bool `json:"continuation,omitempty"`
}

func DefaultLayout() LayoutSettings {
	// Percentage font size is stored as millimeters for project compatibility.
	return LayoutSettings{TemplateName: "新建笔记", LineMode: "fixed", LineCount: 16, PaperWidth: 163, PaperHeight: 240, FontSize: 15 * 1.1, LineHeight: 15, LetterSpacing: 100, LineSpacing: 100, Align: "left"}
}

type MissingGlyph struct {
	Char string  `json:"char"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Size float64 `json:"size"`
}

type FormulaPlacement struct {
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Source string `json:"source"`
	RegionBounds
}

type Page struct {
	Formulas    []FormulaPlacement `json:"formulas,omitempty"`
	Ink         []InkLayer         `json:"ink,omitempty"`
	LineHeight  float64            `json:"lineHeight"`
	LineCount   int                `json:"lineCount"`
	Number      int                `json:"number"`
	Strokes     []Stroke           `json:"strokes"`
	StrokeKinds []string           `json:"strokeKinds,omitempty"`
	TextStrokes []Stroke           `json:"textStrokes"`
	Images      []ImagePlacement   `json:"images"`
	Tables      []TablePlacement   `json:"tables,omitempty"`
	Issues      []string           `json:"issues,omitempty"`
	Warnings    []string           `json:"warnings,omitempty"`
	Missing     []MissingGlyph     `json:"missing"`
	Text        string             `json:"text"`
	Cells       []GlyphCell        `json:"cells"`
	Carets      []CaretStop        `json:"carets"`
	BlankCarets []BlankCaretStop   `json:"blankCarets,omitempty"`
}

type Document struct {
	Erasures       []PageErasure    `json:"erasures,omitempty"`
	ObjectLayout   []ObjectPosition `json:"objectLayout,omitempty"`
	ID             string           `json:"id"`
	Settings       LayoutSettings   `json:"settings"`
	Pages          []Page           `json:"pages"`
	Missing        []string         `json:"missing"`
	CharacterCount int              `json:"characterCount"`
	FontName       string           `json:"fontName"`
}
