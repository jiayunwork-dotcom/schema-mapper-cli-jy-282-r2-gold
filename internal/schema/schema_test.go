package schema

import (
	"strings"
	"testing"
)

func TestParseJSONSchema_Valid(t *testing.T) {
	doc := `{"type":"object","required":["id"],"properties":{"id":{"type":"integer"},"name":{"type":"string"},"tags":{"type":"array","items":{"type":"string"}}}}`
	s, err := ParseJSONSchema(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(s.Fields))
	}
	if s.Fields[0].Name != "id" || s.Fields[0].Type != TInt || s.Fields[0].Optional {
		t.Fatalf("id field wrong: %+v", s.Fields[0])
	}
	if s.Fields[2].Type != TArray || s.Fields[2].Items != TString {
		t.Fatalf("tags field wrong: %+v", s.Fields[2])
	}
}

func TestParseJSONSchema_InvalidJSON(t *testing.T) {
	_, err := ParseJSONSchema(strings.NewReader("{not valid json"))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestParseJSONSchema_UnsupportedType(t *testing.T) {
	doc := `{"type":"object","properties":{"x":{"type":"geometry"}}}`
	_, err := ParseJSONSchema(strings.NewReader(doc))
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestParseDDL_Valid(t *testing.T) {
	ddl := "CREATE TABLE users (id INT NOT NULL, name VARCHAR(64), age INT, active BOOLEAN)"
	s, err := ParseDDL(ddl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "users" {
		t.Fatalf("name = %q", s.Name)
	}
	if len(s.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(s.Fields))
	}
	if s.Fields[0].Type != TInt || s.Fields[0].Optional {
		t.Fatalf("id wrong: %+v", s.Fields[0])
	}
	if s.Fields[1].Type != TString {
		t.Fatalf("name wrong: %+v", s.Fields[1])
	}
	if s.Fields[3].Type != TBool {
		t.Fatalf("active wrong: %+v", s.Fields[3])
	}
}

func TestParseDDL_Malformed(t *testing.T) {
	_, err := ParseDDL("CREATE TABLE bad id INT)")
	if err == nil {
		t.Fatal("expected error for malformed DDL")
	}
}

func TestParseDDL_UnsupportedColumn(t *testing.T) {
	_, err := ParseDDL("CREATE TABLE t (geo GEOMETRY)")
	if err == nil {
		t.Fatal("expected error for unsupported column type")
	}
}

func TestDiff_TypeChangedAddedRemoved(t *testing.T) {
	a := &Schema{Fields: []Field{{Name: "id", Type: TInt}, {Name: "name", Type: TString}}}
	b := &Schema{Fields: []Field{{Name: "id", Type: TString}, {Name: "email", Type: TString}}}
	d := Diff(a, b)
	kinds := map[string]string{}
	for _, c := range d.Changes {
		kinds[c.Field] = c.Kind
	}
	if kinds["id"] != "type_changed" {
		t.Fatalf("id expected type_changed, got %q", kinds["id"])
	}
	if kinds["email"] != "added" {
		t.Fatalf("email expected added, got %q", kinds["email"])
	}
	if kinds["name"] != "removed" {
		t.Fatalf("name expected removed, got %q", kinds["name"])
	}
}

func TestCompatible_LosslessAndMissing(t *testing.T) {
	src := &Schema{Fields: []Field{{Name: "id", Type: TInt}, {Name: "name", Type: TString}}}
	dstOK := &Schema{Fields: []Field{{Name: "id", Type: TFloat}}}
	if ok, _ := Compatible(src, dstOK); !ok {
		t.Fatal("int->float should be compatible")
	}
	dstBad := &Schema{Fields: []Field{{Name: "id", Type: TInt}, {Name: "missing", Type: TString}}}
	ok, reasons := Compatible(src, dstBad)
	if ok {
		t.Fatal("missing required field should be incompatible")
	}
	if len(reasons) == 0 {
		t.Fatal("expected at least one reason")
	}
}

func TestCompatible_MissingOptionalOK(t *testing.T) {
	src := &Schema{Fields: []Field{{Name: "id", Type: TInt}}}
	dst := &Schema{Fields: []Field{
		{Name: "id", Type: TInt},
		{Name: "note", Type: TString, Optional: true},
	}}
	ok, reasons := Compatible(src, dst)
	if !ok {
		t.Fatalf("missing optional target field should be compatible, reasons=%v", reasons)
	}
}
