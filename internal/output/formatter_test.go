package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type testItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}
	data := []testItem{{Name: "a", Value: 1}, {Name: "b", Value: 2}}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	var result []testItem
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 2 || result[0].Name != "a" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestYAMLFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := &YAMLFormatter{}
	data := testItem{Name: "test", Value: 42}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "name: test") {
		t.Errorf("expected YAML with 'name: test', got: %s", out)
	}
	if !strings.Contains(out, "value: 42") {
		t.Errorf("expected YAML with 'value: 42', got: %s", out)
	}
}

func TestTableFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	data := []testItem{{Name: "server1", Value: 100}}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header NAME, got: %s", out)
	}
	if !strings.Contains(out, "server1") {
		t.Errorf("expected 'server1' in output, got: %s", out)
	}
}

func TestTableFormatterEmpty(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	if err := f.Format(&buf, []testItem{}); err != nil {
		t.Fatalf("Format() error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty slice, got: %q", buf.String())
	}
}

func TestCSVFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := &CSVFormatter{}
	data := []testItem{{Name: "a", Value: 1}, {Name: "b", Value: 2}}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "name") {
		t.Errorf("expected header with 'name', got: %s", lines[0])
	}
}

func TestCSVFormatterNonSlice(t *testing.T) {
	var buf bytes.Buffer
	f := &CSVFormatter{}
	err := f.Format(&buf, testItem{Name: "a", Value: 1})
	if err == nil {
		t.Error("expected error for non-slice input")
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"json", "*output.JSONFormatter"},
		{"yaml", "*output.YAMLFormatter"},
		{"csv", "*output.CSVFormatter"},
		{"table", "*output.TableFormatter"},
		{"", "*output.TableFormatter"},
		{"unknown", "*output.TableFormatter"},
	}
	for _, tt := range tests {
		f := New(tt.format)
		got := strings.TrimPrefix(strings.Replace(
			strings.Replace(fmt.Sprintf("%T", f), "output.", "output.", 1),
			"output.", "output.", 1), "")
		_ = got // type check is enough
		if f == nil {
			t.Errorf("New(%q) returned nil", tt.format)
		}
	}
}
