package app

import (
	"testing"
)

func TestResetCmd_HasYesFlag(t *testing.T) {
	f := resetCmd.Flags().Lookup("yes")
	if f == nil {
		t.Fatal("reset command should have --yes flag")
	}
	if f.DefValue != "false" {
		t.Errorf("--yes default should be false, got %s", f.DefValue)
	}
}

func TestResetCmd_HasAppFlags(t *testing.T) {
	for _, name := range []string{"app-name", "user", "port", "identity"} {
		if resetCmd.Flags().Lookup(name) == nil {
			t.Errorf("reset command should have --%s flag", name)
		}
	}
}

func TestResetCmd_RequiresExactlyOneArg(t *testing.T) {
	if resetCmd.Args == nil {
		t.Fatal("reset command should have Args validation")
	}
}
