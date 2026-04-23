package app

import (
	"bytes"
	"strings"
	"testing"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func TestPrintAppList_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := printAppList(&buf, nil); err != nil {
		t.Fatalf("printAppList(nil): %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestPrintAppList_Formatting(t *testing.T) {
	services := []proxypkg.Service{
		{
			Name:  "myapp",
			Hosts: []string{"app.example.com"},
			ActiveTarget: &proxypkg.Target{
				URL: "http://127.0.0.1:34567",
			},
			Phase: proxypkg.Phase("active"),
		},
		{
			Name:  "staging",
			Hosts: []string{"staging.example.com", "alt.example.com"},
			Phase: proxypkg.Phase("pending"),
		},
	}

	var buf bytes.Buffer
	if err := printAppList(&buf, services); err != nil {
		t.Fatalf("printAppList: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"NAME", "PHASE", "ACTIVE", "HOSTS",
		"myapp", "app.example.com", "http://127.0.0.1:34567", "active",
		"staging", "staging.example.com,alt.example.com", "pending",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// No target → "-" placeholder.
	lines := strings.Split(out, "\n")
	foundStaging := false
	for _, l := range lines {
		if strings.HasPrefix(l, "staging") {
			foundStaging = true
			if !strings.Contains(l, "-") {
				t.Errorf("staging row should show '-' for active target: %q", l)
			}
		}
	}
	if !foundStaging {
		t.Error("staging row not found in output")
	}
}
