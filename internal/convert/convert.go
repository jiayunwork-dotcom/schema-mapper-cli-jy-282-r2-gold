// Package convert 提供数据源读取（CSV）与按映射规则批量转换的能力。
package convert

import (
	"encoding/csv"
	"fmt"
	"io"

	"schema-mapper-cli/internal/mapping"
	"schema-mapper-cli/internal/schema"
)

// ReadCSV 读取 CSV，返回记录列表（字段名 -> 值）与表头。
func ReadCSV(r io.Reader) ([]map[string]string, []string, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("csv read: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("empty csv")
	}
	header := rows[0]
	var recs []map[string]string
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		m := map[string]string{}
		for i, h := range header {
			if i < len(row) {
				m[h] = row[i]
			}
		}
		recs = append(recs, m)
	}
	return recs, header, nil
}

// RecordsToTarget 按规则把源记录批量转换为目标记录。
func RecordsToTarget(src []map[string]string, rules []mapping.Rule, dst *schema.Schema) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, 0, len(src))
	for i, rec := range src {
		row, err := mapping.Apply(rec, rules, dst)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i, err)
		}
		out = append(out, row)
	}
	return out, nil
}
