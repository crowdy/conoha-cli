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
			Phase: proxypkg.PhaseLive,
		},
		{
			Name:  "staging",
			Hosts: []string{"staging.example.com", "alt.example.com"},
			Phase: proxypkg.PhaseConfigured,
		},
	}

	var buf bytes.Buffer
	if err := printAppList(&buf, services); err != nil {
		t.Fatalf("printAppList: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"NAME", "PHASE", "ACTIVE", "HOSTS",
		"myapp", "app.example.com", "http://127.0.0.1:34567", "live",
		"staging", "staging.example.com,alt.example.com", "configured",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	// Row-shape check: myapp row must have PHASE=live positioned between
	// the name and the active URL (catches a column-swap bug that whole-
	// buffer Contains would miss).
	lines := strings.Split(out, "\n")
	var myappRow, stagingRow string
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "myapp"):
			myappRow = l
		case strings.HasPrefix(l, "staging"):
			stagingRow = l
		}
	}
	if myappRow == "" {
		t.Fatal("myapp row not found")
	}
	if stagingRow == "" {
		t.Fatal("staging row not found")
	}

	// PHASE column must sit between NAME and ACTIVE in the myapp row.
	idxName := strings.Index(myappRow, "myapp")
	idxPhase := strings.Index(myappRow, "live")
	idxActive := strings.Index(myappRow, "http://")
	if !(idxName < idxPhase && idxPhase < idxActive) {
		t.Errorf("column order broken in myapp row: %q", myappRow)
	}

	// No target → "-" placeholder in ACTIVE column.
	if !strings.Contains(stagingRow, " - ") {
		t.Errorf("staging row should show '-' for active target: %q", stagingRow)
	}
}
