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
	if err := resetCmd.Args(resetCmd, []string{}); err == nil {
		t.Error("should reject zero args")
	}
	if err := resetCmd.Args(resetCmd, []string{"server1"}); err != nil {
		t.Errorf("should accept one arg: %v", err)
	}
	if err := resetCmd.Args(resetCmd, []string{"server1", "server2"}); err == nil {
		t.Error("should reject two args")
	}
}
