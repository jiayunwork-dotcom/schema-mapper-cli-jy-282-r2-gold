package mapping

import (
	"testing"

	"schema-mapper-cli/internal/schema"
)

func TestLevenshtein_Distances(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"user_name", "username", 1},
		{"id", "id", 0},
	}
	for _, c := range cases {
		if got := Levenshtein(c.a, c.b); got != c.want {
			t.Errorf("Levenshtein(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSuggest_MapsSimilarFields(t *testing.T) {
	src := &schema.Schema{Fields: []schema.Field{
		{Name: "userid", Type: schema.TInt},
		{Name: "fullname", Type: schema.TString},
		{Name: "emailaddr", Type: schema.TString},
	}}
	dst := &schema.Schema{Fields: []schema.Field{
		{Name: "username", Type: schema.TInt},
		{Name: "name", Type: schema.TString},
	}}
	rules := Suggest(src, dst, 4)
	got := map[string]string{}
	for _, r := range rules {
		got[r.Target] = r.Source
	}
	if got["username"] != "userid" {
		t.Errorf("username should map to userid, got %q", got["username"])
	}
	if got["name"] != "fullname" {
		t.Errorf("name should map to fullname, got %q", got["name"])
	}
}

func TestApply_TypeConversionError(t *testing.T) {
	dst := &schema.Schema{Fields: []schema.Field{{Name: "age", Type: schema.TInt}}}
	rules := []Rule{{Source: "age", Target: "age"}}
	_, err := Apply(map[string]string{"age": "notanumber"}, rules, dst)
	if err == nil {
		t.Fatal("expected error converting non-numeric value to int")
	}
}

func TestApply_DefaultForMissing(t *testing.T) {
	dst := &schema.Schema{Fields: []schema.Field{{Name: "country", Type: schema.TString, Optional: true}}}
	rules := []Rule{{Source: "country", Target: "country", Default: "CN"}}
	out, err := Apply(map[string]string{}, rules, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["country"] != "CN" {
		t.Fatalf("expected default CN, got %v", out["country"])
	}
}

func TestApply_MissingRequiredFails(t *testing.T) {
	dst := &schema.Schema{Fields: []schema.Field{{Name: "id", Type: schema.TInt}}}
	rules := []Rule{{Source: "id", Target: "id"}}
	_, err := Apply(map[string]string{}, rules, dst)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
}

func TestSuggest_ResultsIndependent(t *testing.T) {
	src1 := &schema.Schema{Fields: []schema.Field{{Name: "userid", Type: schema.TInt}}}
	dst1 := &schema.Schema{Fields: []schema.Field{{Name: "username", Type: schema.TInt}}}
	first := Suggest(src1, dst1, 4)
	if len(first) != 1 || first[0].Source != "userid" || first[0].Target != "username" {
		t.Fatalf("first suggestion: %+v", first)
	}

	src2 := &schema.Schema{Fields: []schema.Field{{Name: "emailaddr", Type: schema.TString}}}
	dst2 := &schema.Schema{Fields: []schema.Field{{Name: "email", Type: schema.TString}}}
	second := Suggest(src2, dst2, 4)
	if len(second) != 1 {
		t.Fatalf("second suggestion: %+v", second)
	}

	if first[0].Source != "userid" || first[0].Target != "username" {
		t.Fatalf("first suggestion changed after second call: %+v", first)
	}
}

func TestApply_EmptyStringUsesDefault(t *testing.T) {
	dst := &schema.Schema{Fields: []schema.Field{{Name: "country", Type: schema.TString}}}
	rules := []Rule{{Source: "country", Target: "country", Default: "CN"}}
	out, err := Apply(map[string]string{"country": ""}, rules, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["country"] != "CN" {
		t.Fatalf("empty source should use default, got %v", out["country"])
	}
}
