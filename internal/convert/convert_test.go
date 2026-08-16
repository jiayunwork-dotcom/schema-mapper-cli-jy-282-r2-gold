package convert

import (
	"strings"
	"testing"

	"schema-mapper-cli/internal/mapping"
	"schema-mapper-cli/internal/schema"
)

func TestReadCSV_AndConvert(t *testing.T) {
	csvData := "user_id,name\n1,alice\n2,bob\n"
	recs, header, err := ReadCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(header) != 2 || header[0] != "user_id" {
		t.Fatalf("header wrong: %v", header)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
	dst := &schema.Schema{Fields: []schema.Field{{Name: "id", Type: schema.TInt}, {Name: "name", Type: schema.TString}}}
	rules := []mapping.Rule{{Source: "user_id", Target: "id"}, {Source: "name", Target: "name"}}
	out, err := RecordsToTarget(recs, rules, dst)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}
	if out[0]["id"] != 1 || out[1]["name"] != "bob" {
		t.Fatalf("converted wrong: %v", out)
	}
}

func TestReadCSV_Empty(t *testing.T) {
	_, _, err := ReadCSV(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty csv")
	}
}

func TestRecordsToTarget_BadRowFails(t *testing.T) {
	recs := []map[string]string{{"age": "12"}, {"age": "x"}}
	dst := &schema.Schema{Fields: []schema.Field{{Name: "age", Type: schema.TInt}}}
	rules := []mapping.Rule{{Source: "age", Target: "age"}}
	_, err := RecordsToTarget(recs, rules, dst)
	if err == nil {
		t.Fatal("expected error when a row cannot convert")
	}
}

func TestRecordsToTarget_EmptyNonNil(t *testing.T) {
	dst := &schema.Schema{Fields: []schema.Field{{Name: "id", Type: schema.TInt}}}
	out, err := RecordsToTarget(nil, nil, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("empty input should return an empty slice, not nil")
	}
	if len(out) != 0 {
		t.Fatalf("len=%d", len(out))
	}
}
