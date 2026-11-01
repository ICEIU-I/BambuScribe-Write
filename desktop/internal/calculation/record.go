package calculation

import "fmt"

// Record retains editable readings and rules alongside the generated body text.
// A changed or deleted text block is detached, never silently regenerated.
type Record struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Plan Plan   `json:"plan"`
}

func ValidateRecords(records []Record) error {
	if len(records) > 100 {
		return fmt.Errorf("数据处理组过多，请拆分项目")
	}
	ids := map[string]bool{}
	for _, r := range records {
		if r.ID == "" || len(r.ID) > 100 || ids[r.ID] || len(r.Text) > 120000 {
			return fmt.Errorf("数据处理记录无效")
		}
		ids[r.ID] = true
		if err := r.Plan.Validate(); err != nil {
			return err
		}
	}
	return nil
}
