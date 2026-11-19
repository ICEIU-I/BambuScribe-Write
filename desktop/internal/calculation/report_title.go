package calculation

import (
	"regexp"
	"strings"
)

// These prefixes describe the generation process, not the experiment. Keep
// provenance in Variable.Source; only titles enter the printed report here.
var reportProcessPrefix = regexp.MustCompile(`^(?:按照|根据|依据|参照|按)(?:参考样例|参考示例|参考图片|参考方法|参考步骤|参考|样例|示例|用户要求|你的要求|要求|所给图片|图片|识别结果)`)
var reportProcessAside = regexp.MustCompile(`[（(](?:按照|根据|依据|参照|按)(?:参考样例|参考示例|参考图片|参考|样例|示例|用户要求|要求)[）)]`)

func cleanReportTitle(title string) string {
	title = reportProcessAside.ReplaceAllString(title, "")
	parts := strings.FieldsFunc(title, func(r rune) bool { return r == '；' || r == ';' })
	kept := make([]string, 0, len(parts))
	changed := false
	for _, part := range parts {
		text := strings.TrimSpace(part)
		prefix := reportProcessPrefix.FindString(text)
		// "参考值" is a scientific quantity, not an instruction to the model.
		if prefix != "" && !strings.HasPrefix(text[len(prefix):], "值") {
			changed = true
			if len(kept) > 0 {
				continue // Drop an appended process note after a complete title.
			}
			text = strings.TrimLeft(text[len(prefix):], " ：:，,、")
		}
		if text != "" {
			kept = append(kept, text)
		}
	}
	if !changed {
		return title
	}
	return strings.Join(kept, "；")
}

// CleanReportTitles also covers imported plans. Calculate repeats this narrow
// cleanup for saved plans, so recalculating an old record cannot restore notes.
func (p Plan) CleanReportTitles() Plan {
	p.Sections = append([]Section(nil), p.Sections...)
	for i := range p.Sections {
		p.Sections[i].Title = cleanReportTitle(p.Sections[i].Title)
	}
	return p
}
