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
