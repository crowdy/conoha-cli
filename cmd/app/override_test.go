package app

import (
	"strings"
	"testing"
)

func TestComposeOverride_WebPortAndName(t *testing.T) {
	got := composeOverride("myapp", "a1b2c3d", "web", 8080)
	want := []string{
		`services:`,
		`  web:`,
		`    container_name: myapp-a1b2c3d-web`,
		`    ports:`,
		`      - "127.0.0.1:0:8080"`,
	}
	for _, line := range want {
		if !strings.Contains(got, line) {
			t.Errorf("missing line %q in:\n%s", line, got)
		}
	}
}

func TestAccessoryProjectName(t *testing.T) {
	if got := accessoryProjectName("myapp"); got != "myapp-accessories" {
		t.Errorf("got %q", got)
	}
}

func TestSlotProjectName(t *testing.T) {
	if got := slotProjectName("myapp", "a1b2c3d"); got != "myapp-a1b2c3d" {
		t.Errorf("got %q", got)
	}
}
