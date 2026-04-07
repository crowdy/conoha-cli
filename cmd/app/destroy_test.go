package app

import (
	"testing"
)

func TestDestroyCmd_HasYesFlag(t *testing.T) {
	f := destroyCmd.Flags().Lookup("yes")
	if f == nil {
		t.Fatal("destroy command should have --yes flag")
	}
	if f.DefValue != "false" {
		t.Errorf("--yes default should be false, got %s", f.DefValue)
	}
}
