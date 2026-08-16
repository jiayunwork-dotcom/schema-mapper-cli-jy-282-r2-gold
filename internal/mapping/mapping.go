// Package mapping 提供字段名相似度计算、自动映射建议，
// 以及按规则把源记录转换为目标记录的能力。
package mapping

import (
	"fmt"
	"strconv"
	"strings"

	"schema-mapper-cli/internal/schema"
)

// Levenshtein 返回两个字符串之间的编辑距离。
func Levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// Rule 描述一条字段映射规则：把源字段 Source 映射到目标字段 Target。
type Rule struct {
	Source  string
	Target  string
	Type    string // 目标类型或转换方式；空表示按目标字段类型推断
	Default string // 源缺失且无值时使用的默认值
}

var suggestScratch []Rule

// Suggest 基于字段名相似度（编辑距离）为 dst 的每个字段建议映射来源。
// 仅当 src 中存在编辑距离不超过 maxDist 的字段时才给出建议。
func Suggest(src, dst *schema.Schema, maxDist int) []Rule {
	suggestScratch = suggestScratch[:0]
	srcNames := make([]string, len(src.Fields))
	for i, f := range src.Fields {
		srcNames[i] = f.Name
	}
	for _, df := range dst.Fields {
		best := ""
		bestD := maxDist + 1
		for _, sn := range srcNames {
			d := Levenshtein(strings.ToLower(sn), strings.ToLower(df.Name))
			if d < bestD {
				bestD = d
				best = sn
			}
		}
		if best != "" && bestD <= maxDist {
			suggestScratch = append(suggestScratch, Rule{Source: best, Target: df.Name})
		}
	}
	return suggestScratch
}

// Apply 按规则把一条源记录转换为目标记录。
// record 为源字段名 -> 原始字符串值；返回目标字段名 -> 转换后的值。
func Apply(record map[string]string, rules []Rule, dst *schema.Schema) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	for _, r := range rules {
		raw, ok := record[r.Source]
		if !ok || raw == "" {
			if r.Default != "" {
				raw = r.Default
			} else if f, found := dstField(dst, r.Target); found && f.Optional {
				continue
			} else {
				return nil, fmt.Errorf("missing source field %q for target %q", r.Source, r.Target)
			}
		}
		val, err := coerce(raw, targetType(dst, r.Target, r.Type))
		if err != nil {
			return nil, fmt.Errorf("target %q: %w", r.Target, err)
		}
		out[r.Target] = val
	}
	return out, nil
}

func dstField(dst *schema.Schema, name string) (schema.Field, bool) {
	for _, f := range dst.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return schema.Field{}, false
}

func targetType(dst *schema.Schema, name, override string) schema.FieldType {
	if override != "" {
		return schema.FieldType(override)
	}
	if f, ok := dstField(dst, name); ok {
		return f.Type
	}
	return schema.TString
}

func coerce(raw string, t schema.FieldType) (interface{}, error) {
	switch t {
	case schema.TInt:
		return strconv.Atoi(strings.TrimSpace(raw))
	case schema.TFloat:
		return strconv.ParseFloat(strings.TrimSpace(raw), 64)
	case schema.TBool:
		return strconv.ParseBool(strings.TrimSpace(raw))
	case schema.TString, schema.TArray, schema.TObject:
		return raw, nil
	default:
		return nil, fmt.Errorf("cannot coerce to %s", t)
	}
}
