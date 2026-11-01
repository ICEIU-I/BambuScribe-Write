package application

import "bambuscribe/desktop/internal/engine"

// Help examples use the production layout engine without entering the user's
// export cache: browsing categories must never evict their current document.
func (a *App) PreviewWritingRule(text, fontID string) (*engine.Document, error) {
	if a.fontError != nil {
		return nil, a.fontError
	}
	s := engine.DefaultLayout()
	s.FontID, s.FontSize = fontID, 6
	s.PaperWidth, s.PaperHeight, s.LineCount = 160, 80, 5
	s.MarginLeft, s.MarginRight, s.MarginTop, s.MarginBottom = 3, 3, 3, 3
	return a.font.Layout(text, s)
}
