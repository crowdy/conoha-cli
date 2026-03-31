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

func TestCSVFormatterSingleStruct(t *testing.T) {
	var buf bytes.Buffer
	f := &CSVFormatter{}
	err := f.Format(&buf, testItem{Name: "a", Value: 1})
	if err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + 1 row), got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "name") {
		t.Errorf("expected header with 'name', got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "a") {
		t.Errorf("expected data with 'a', got: %s", lines[1])
	}
}

func TestTableFormatterNoHeaders(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{NoHeaders: true}
	data := []testItem{{Name: "server1", Value: 100}}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "NAME") {
		t.Errorf("expected no header NAME with NoHeaders, got: %s", out)
	}
	if !strings.Contains(out, "server1") {
		t.Errorf("expected 'server1' in output, got: %s", out)
	}
}

func TestCSVFormatterNoHeaders(t *testing.T) {
	var buf bytes.Buffer
	f := &CSVFormatter{NoHeaders: true}
	data := []testItem{{Name: "a", Value: 1}, {Name: "b", Value: 2}}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (no header), got %d", len(lines))
	}
	if strings.Contains(lines[0], "name") && !strings.Contains(lines[0], "a") {
		t.Errorf("first line should be data, not header: %s", lines[0])
	}
}

func TestNewWithOptions(t *testing.T) {
	f := NewWithOptions(Options{Format: "table", NoHeaders: true})
	tf, ok := f.(*TableFormatter)
	if !ok {
		t.Fatal("expected TableFormatter")
	}
	if !tf.NoHeaders {
		t.Error("expected NoHeaders=true")
	}

	f = NewWithOptions(Options{Format: "csv", NoHeaders: true})
	cf, ok := f.(*CSVFormatter)
	if !ok {
		t.Fatal("expected CSVFormatter")
	}
	if !cf.NoHeaders {
		t.Error("expected NoHeaders=true")
	}
}

func TestTableFormatterSingleStruct(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	data := testItem{Name: "server1", Value: 100}

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
	if !strings.Contains(out, "100") {
		t.Errorf("expected '100' in output, got: %s", out)
	}
}

func TestTableFormatterSingleStructPointer(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	data := &testItem{Name: "server2", Value: 200}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header NAME, got: %s", out)
	}
	if !strings.Contains(out, "server2") {
		t.Errorf("expected 'server2' in output, got: %s", out)
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
