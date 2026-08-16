// Package schema 提供异构数据源 Schema 的统一内部表示（IR），
// 以及从 JSON Schema / SQL DDL / CSV 表头解析、对比与兼容性检查的能力。
package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// FieldType 是统一内部表示中的字段类型。
type FieldType string

const (
	TString FieldType = "string"
	TInt    FieldType = "int"
	TFloat  FieldType = "float"
	TBool   FieldType = "bool"
	TArray  FieldType = "array"
	TObject FieldType = "object"
)

// Field 是统一内部表示中的单个字段。
type Field struct {
	Name     string
	Type     FieldType
	Optional bool
	Items    FieldType // 仅当 Type==TArray 时有效，表示元素类型
	Default  string
}

// Schema 是多个字段的统一内部表示。
type Schema struct {
	Name   string
	Fields []Field
}

// ParseJSONSchema 从简化的 JSON Schema（object 根）解析出统一内部表示。
func ParseJSONSchema(r io.Reader) (*Schema, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("invalid json schema: %w", err)
	}
	if strings.TrimSpace(doc.Type) != "" && doc.Type != "object" {
		return nil, fmt.Errorf("only object root is supported, got %q", doc.Type)
	}
	req := map[string]bool{}
	for _, k := range doc.Required {
		req[k] = true
	}
	names := make([]string, 0, len(doc.Properties))
	for n := range doc.Properties {
		names = append(names, n)
	}
	sort.Strings(names)
	s := &Schema{}
	for _, n := range names {
		ft, optional, items, err := parseProperty(doc.Properties[n], req[n])
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", n, err)
		}
		s.Fields = append(s.Fields, Field{Name: n, Type: ft, Optional: optional, Items: items})
	}
	return s, nil
}

func parseProperty(raw json.RawMessage, required bool) (FieldType, bool, FieldType, error) {
	var t struct {
		Type  string `json:"type"`
		Items struct {
			Type string `json:"type"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		return "", false, "", err
	}
	ft, items, err := normalizeType(t.Type, t.Items.Type)
	if err != nil {
		return "", false, "", err
	}
	return ft, !required, items, nil
}

func normalizeType(t, items string) (FieldType, FieldType, error) {
	switch t {
	case "string":
		return TString, "", nil
	case "integer":
		return TInt, "", nil
	case "number":
		return TFloat, "", nil
	case "boolean":
		return TBool, "", nil
	case "array":
		it, _, err := normalizeType(items, "")
		if err != nil {
			return "", "", err
		}
		return TArray, it, nil
	case "object":
		return TObject, "", nil
	case "":
		return "", "", fmt.Errorf("missing type")
	default:
		return "", "", fmt.Errorf("unsupported type %q", t)
	}
}

// ParseCSVHeader 从 CSV 表头行推断 schema（全部字段为 string，均必填）。
func ParseCSVHeader(header []string) *Schema {
	s := &Schema{}
	for _, h := range header {
		s.Fields = append(s.Fields, Field{Name: h, Type: TString, Optional: false})
	}
	return s
}

// ParseDDL 从简化的 CREATE TABLE 语句解析 schema。
func ParseDDL(s string) (*Schema, error) {
	open := strings.Index(s, "(")
	close := strings.LastIndex(s, ")")
	if open < 0 || close < 0 || close < open {
		return nil, fmt.Errorf("malformed CREATE TABLE: missing parentheses")
	}
	name := strings.TrimSpace(s[:open])
	if i := strings.LastIndex(name, " "); i >= 0 {
		name = name[i+1:]
	}
	body := s[open+1 : close]
	cols := splitColumns(body)
	sch := &Schema{Name: name}
	for _, c := range cols {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		parts := strings.Fields(c)
		if len(parts) < 2 {
			return nil, fmt.Errorf("malformed column definition: %q", c)
		}
		colName := strings.Trim(parts[0], "`\"")
		colType := strings.ToUpper(strings.TrimSuffix(parts[1], ";"))
		ft, err := ddlTypeToField(colType)
		if err != nil {
			return nil, fmt.Errorf("column %q: %w", colName, err)
		}
		optional := !strings.Contains(strings.ToUpper(c), "NOT NULL")
		sch.Fields = append(sch.Fields, Field{Name: colName, Type: ft, Optional: optional})
	}
	return sch, nil
}

func splitColumns(body string) []string {
	var cols []string
	var depth int
	var cur strings.Builder
	for _, r := range body {
		switch r {
		case '(':
			depth++
			cur.WriteRune(r)
		case ')':
			depth--
			cur.WriteRune(r)
		case ',':
			if depth == 0 {
				cols = append(cols, cur.String())
				cur.Reset()
			} else {
				cur.WriteRune(r)
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		cols = append(cols, cur.String())
	}
	return cols
}

func ddlTypeToField(t string) (FieldType, error) {
	switch {
	case strings.HasPrefix(t, "INT"), t == "BIGINT", t == "SMALLINT", t == "SERIAL":
		return TInt, nil
	case strings.HasPrefix(t, "VARCHAR"), strings.HasPrefix(t, "CHAR"), t == "TEXT", t == "UUID", t == "DATE", t == "TIMESTAMP":
		return TString, nil
	case strings.HasPrefix(t, "DECIMAL"), t == "NUMERIC", t == "FLOAT", t == "DOUBLE", t == "REAL":
		return TFloat, nil
	case t == "BOOLEAN", t == "BOOL":
		return TBool, nil
	default:
		return "", fmt.Errorf("unsupported column type %q", t)
	}
}

// FieldChange 描述两个 schema 之间单个字段的变化。
type FieldChange struct {
	Field string
	Kind  string // added | removed | type_changed | optional_changed
	From  string
	To    string
}

// SchemaDiff 是两个 schema 之间所有字段变化的集合。
type SchemaDiff struct {
	Changes []FieldChange
}

// Diff 对比两个 schema，返回新增 / 删除 / 类型变更 / 可选性变更。
func Diff(a, b *Schema) SchemaDiff {
	am := map[string]Field{}
	for _, f := range a.Fields {
		am[f.Name] = f
	}
	bm := map[string]Field{}
	for _, f := range b.Fields {
		bm[f.Name] = f
	}
	var d SchemaDiff
	for name, bf := range bm {
		af, ok := am[name]
		if !ok {
			d.Changes = append(d.Changes, FieldChange{Field: name, Kind: "added", To: string(bf.Type)})
			continue
		}
		if af.Type != bf.Type {
			d.Changes = append(d.Changes, FieldChange{Field: name, Kind: "type_changed", From: string(af.Type), To: string(bf.Type)})
		}
		if af.Optional != bf.Optional {
			d.Changes = append(d.Changes, FieldChange{Field: name, Kind: "optional_changed", From: boolStr(af.Optional), To: boolStr(bf.Optional)})
		}
		_ = af
	}
	for name, af := range am {
		if _, ok := bm[name]; !ok {
			d.Changes = append(d.Changes, FieldChange{Field: name, Kind: "removed", From: string(af.Type)})
		}
	}
	sort.Slice(d.Changes, func(i, j int) bool { return d.Changes[i].Field < d.Changes[j].Field })
	return d
}

func boolStr(b bool) string {
	if b {
		return "optional"
	}
	return "required"
}

// Compatible 检查源 schema a 的数据能否无损转换为目标 schema b。
func Compatible(a, b *Schema) (bool, []string) {
	am := map[string]Field{}
	for _, f := range a.Fields {
		am[f.Name] = f
	}
	var reasons []string
	for _, bf := range b.Fields {
		af, ok := am[bf.Name]
		if !ok {
			if !bf.Optional {
				reasons = append(reasons, fmt.Sprintf("target field %q missing in source and not optional", bf.Name))
			}
			continue
		}
		if !typeLossless(af.Type, bf.Type) {
			reasons = append(reasons, fmt.Sprintf("type %s->%s for %q may lose information", af.Type, bf.Type, bf.Name))
		}
	}
	return len(reasons) == 0, reasons
}

func typeLossless(from, to FieldType) bool {
	if from == to {
		return true
	}
	switch from {
	case TInt:
		return to == TFloat || to == TString
	case TFloat, TBool:
		return to == TString
	default:
		return false
	}
}
