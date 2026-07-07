package api

import "testing"

func TestMaskSensitive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password in JSON body",
			input:    `{"user":{"name":"admin","password":"secret123"}}`,
			expected: `{"user":{"name":"admin","password":"****"}}`,
		},
		{
			name:     "no password",
			input:    `{"name":"test"}`,
			expected: `{"name":"test"}`,
		},
		{
			name:     "password with spaces",
			input:    `{"password" : "my pass"}`,
			expected: `{"password":"****"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskSensitive(tt.input)
			if got != tt.expected {
				t.Errorf("maskSensitive(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDebugLevelFromEnv(t *testing.T) {
	// Save and restore
	origLevel := debugLevel
	defer func() { debugLevel = origLevel }()

	// Test SetDebugLevel only increases
	debugLevel = DebugOff
	SetDebugLevel(DebugVerbose)
	if debugLevel != DebugVerbose {
		t.Errorf("expected DebugVerbose, got %d", debugLevel)
	}

	SetDebugLevel(DebugOff) // should not decrease
	if debugLevel != DebugVerbose {
		t.Errorf("expected DebugVerbose (not decreased), got %d", debugLevel)
	}

	SetDebugLevel(DebugAPI)
	if debugLevel != DebugAPI {
		t.Errorf("expected DebugAPI, got %d", debugLevel)
	}
}

func TestFormatBodyPrettyPrintsJSON(t *testing.T) {
	// A JSON body must be indented and every line prefixed, so a large payload
	// renders as many short, wrapping-friendly lines instead of one giant line
	// whose start scrolls off-screen.
	got := formatBody("< ", []byte(`{"a":1,"password":"secret"}`))
	want := "< {\n<   \"a\": 1,\n<   \"password\": \"****\"\n< }\n"
	if got != want {
		t.Errorf("formatBody() =\n%q\nwant\n%q", got, want)
	}
}

func TestFormatBodyNonJSONFallsBack(t *testing.T) {
	// Non-JSON bodies (e.g. an HTML error page) are printed verbatim, one
	// prefixed line, never dropped.
	got := formatBody("> ", []byte("not json at all"))
	want := "> not json at all\n"
	if got != want {
		t.Errorf("formatBody() = %q, want %q", got, want)
	}
}

func TestFormatBodyIndentsJSONArrayRoot(t *testing.T) {
	// A top-level JSON array (as returned by several list endpoints) is
	// indented, not left on one line.
	got := formatBody("< ", []byte(`[{"id":"x"}]`))
	want := "< [\n<   {\n<     \"id\": \"x\"\n<   }\n< ]\n"
	if got != want {
		t.Errorf("formatBody() =\n%q\nwant\n%q", got, want)
	}
}

func TestFormatBodyTrailingNewlineNoStrayLine(t *testing.T) {
	// A non-JSON body ending in a newline must not produce a trailing
	// prefix-only line (e.g. a bare "< ").
	got := formatBody("< ", []byte("<html>error</html>\n"))
	want := "< <html>error</html>\n"
	if got != want {
		t.Errorf("formatBody() = %q, want %q", got, want)
	}
}

func TestFormatBodyMultiLineNonJSON(t *testing.T) {
	// Every line of a multi-line non-JSON body is prefixed.
	got := formatBody("> ", []byte("line1\nline2"))
	want := "> line1\n> line2\n"
	if got != want {
		t.Errorf("formatBody() = %q, want %q", got, want)
	}
}

func TestSensitiveHeaders(t *testing.T) {
	if !sensitiveHeaders["X-Auth-Token"] {
		t.Error("X-Auth-Token should be sensitive")
	}
	if !sensitiveHeaders["X-Subject-Token"] {
		t.Error("X-Subject-Token should be sensitive")
	}
	if !sensitiveHeaders["Authorization"] {
		t.Error("Authorization should be sensitive")
	}
	if sensitiveHeaders["Content-Type"] {
		t.Error("Content-Type should not be sensitive")
	}
}
