package api

import (
	"strings"
	"testing"
)

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
		{
			name:     "keypair private_key",
			input:    `{"keypair":{"name":"k","private_key":"-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END-----\n"}}`,
			expected: `{"keypair":{"name":"k","private_key":"****"}}`,
		},
		{
			name:     "server adminPass",
			input:    `{"server":{"id":"x","adminPass":"hunter2"}}`,
			expected: `{"server":{"id":"x","adminPass":"****"}}`,
		},
		{
			name:     "application credential secret",
			input:    `{"application_credential":{"secret":"s3cr3t"}}`,
			expected: `{"application_credential":{"secret":"****"}}`,
		},
		{
			name:     "password value containing an escaped quote",
			input:    `{"password":"a\"b","name":"keep"}`,
			expected: `{"password":"****","name":"keep"}`,
		},
		{
			name:     "similarly named key is not masked",
			input:    `{"password_hash":"keep","secret_ref":"keep"}`,
			expected: `{"password_hash":"keep","secret_ref":"keep"}`,
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

func TestFormatBodyMasksPrivateKeyInPrettyPrint(t *testing.T) {
	// A Nova-generated keypair response carries the private key in the body;
	// formatBody must mask it even while pretty-printing.
	body := []byte(`{"keypair":{"name":"k","private_key":"-----BEGIN KEY-----\nsecret\n-----END-----\n"}}`)
	got := formatBody("< ", body)
	if strings.Contains(got, "BEGIN KEY") || strings.Contains(got, "secret") {
		t.Errorf("private key leaked in debug output:\n%s", got)
	}
	if !strings.Contains(got, `"private_key": "****"`) {
		t.Errorf("expected masked private_key in output:\n%s", got)
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
