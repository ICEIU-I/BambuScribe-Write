package application

import (
	"bambuscribe/desktop/internal/project"
	"strings"
	"unicode"
	"unicode/utf8"
)

func defaultProjectFilename(text string) string {
	if end := strings.IndexAny(text, "\r\n"); end >= 0 {
		text = text[:end]
	}
	var name strings.Builder
	for _, character := range strings.TrimSpace(text) {
		if unicode.IsControl(character) {
			continue
		}
		if strings.ContainsRune(`/\:*?"<>|`, character) {
			character = '_'
		}
		// Leave room for the extension within the filesystem's byte limit.
		if name.Len()+utf8.RuneLen(character) > 240 {
			break
		}
		name.WriteRune(character)
	}
	base := strings.TrimSpace(name.String())
	if base == "" {
		base = "手写项目"
	}
	if strings.HasSuffix(strings.ToLower(base), project.Extension) {
		return base
	}
	return base + project.Extension
}
