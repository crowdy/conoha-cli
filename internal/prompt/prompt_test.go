package prompt

import (
	"os"
	"testing"
)

func TestString_NoInput(t *testing.T) {
	t.Setenv("CONOHA_NO_INPUT", "1")
	val, err := String("Enter value")
	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}
	if err == nil {
		t.Fatal("expected error when CONOHA_NO_INPUT=1")
	}
	if got := err.Error(); got != "input required but --no-input is set" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestConfirm_Yes_Env(t *testing.T) {
	t.Setenv("CONOHA_YES", "1")
	ok, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected true when CONOHA_YES=1")
	}
}

func TestConfirm_NoInput_Without_Yes(t *testing.T) {
	t.Setenv("CONOHA_NO_INPUT", "1")
	ok, err := Confirm("Delete?")
	if ok {
		t.Fatal("expected false when --no-input without --yes")
	}
	if err == nil {
		t.Fatal("expected error when --no-input without --yes")
	}
	if got := err.Error(); got != "confirmation required but --no-input is set; use --yes to auto-confirm" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestConfirm_Interactive_Yes(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.WriteString("y\n")
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	ok, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected true for 'y' input")
	}
}

func TestConfirm_Interactive_No(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.WriteString("n\n")
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	ok, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected false for 'n' input")
	}
}

func TestConfirm_Interactive_Empty_Default_No(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.WriteString("\n")
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	ok, err := Confirm("Delete?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected false for empty input (default no)")
	}
}
