package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestSummary_CoverageAndText(t *testing.T) {
	s := &Summary{MappedFields: 3, TotalFields: 4, Warnings: []string{"x"}}
	if s.Coverage() < 0.74 || s.Coverage() > 0.76 {
		t.Fatalf("coverage = %v", s.Coverage())
	}
	var buf bytes.Buffer
	s.RenderText(&buf)
	if !strings.Contains(buf.String(), "75.0%") {
		t.Fatalf("text missing coverage: %q", buf.String())
	}
}

func TestSummary_JSON(t *testing.T) {
	s := &Summary{MappedFields: 1, TotalFields: 1}
	var buf bytes.Buffer
	if err := s.RenderJSON(&buf); err != nil {
		t.Fatalf("json error: %v", err)
	}
	if !strings.Contains(buf.String(), "\"mapped_fields\": 1") {
		t.Fatalf("json wrong: %q", buf.String())
	}
}
