package engine

// LayoutDiagnostic locates an editing issue on the paper. It is preview
// metadata; it never contributes strokes to the writing/export stream.
type LayoutDiagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	RegionBounds
}

func appendOverlapDiagnostics(page *Page, bounds []RegionBounds, seen map[RegionBounds]bool) {
	for _, b := range bounds {
		if seen[b] {
			continue
		}
		seen[b] = true
		page.Diagnostics = append(page.Diagnostics, LayoutDiagnostic{
			Code: "line-overlap", Message: lineOverlapIssue, RegionBounds: b,
		})
	}
}
