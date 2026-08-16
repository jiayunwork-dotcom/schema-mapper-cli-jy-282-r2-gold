// Package report 提供映射转换的汇总报告生成（文本 / JSON）。
package report

import (
	"encoding/json"
	"fmt"
	"io"
)

// Summary 是映射转换的汇总报告。
type Summary struct {
	MappedFields int      `json:"mapped_fields"`
	TotalFields  int      `json:"total_fields"`
	Warnings     []string `json:"warnings"`
}

// Coverage 返回映射覆盖率（0~1）。
func (s *Summary) Coverage() float64 {
	if s.TotalFields == 0 {
		return 0
	}
	return float64(s.MappedFields) / float64(s.TotalFields)
}

// RenderText 以可读文本形式输出报告。
func (s *Summary) RenderText(w io.Writer) {
	fmt.Fprintf(w, "mapped fields : %d / %d\n", s.MappedFields, s.TotalFields)
	fmt.Fprintf(w, "coverage      : %.1f%%\n", s.Coverage()*100)
	for _, warn := range s.Warnings {
		fmt.Fprintf(w, "warning       : %s\n", warn)
	}
}

// RenderJSON 以 JSON 形式输出报告。
func (s *Summary) RenderJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}
