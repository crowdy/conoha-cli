package app

import (
	"strings"
	"testing"
)

func TestComposeOverride_WebPortAndName_NoAccessories(t *testing.T) {
	got := composeOverride("myapp", "a1b2c3d", "web", 8080, false)
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
	if strings.Contains(got, "networks:") {
		t.Errorf("no-accessories form should not emit networks: block:\n%s", got)
	}
}

func TestComposeOverride_WithAccessoriesJoinsExternalNetwork(t *testing.T) {
	got := composeOverride("myapp", "a1b2c3d", "web", 8080, true)
	want := []string{
		`container_name: myapp-a1b2c3d-web`,
		`    networks:`,
		`      - default`,
		`      - accessories`,
		`networks:`,
		`  accessories:`,
		`    name: myapp-accessories_default`,
		`    external: true`,
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
