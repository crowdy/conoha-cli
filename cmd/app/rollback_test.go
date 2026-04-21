package app

import (
	"strings"
	"testing"
)

func TestRollbackCmd_HasModeFlags(t *testing.T) {
	if rollbackCmd.Flags().Lookup("proxy") == nil {
		t.Error("rollback should have --proxy flag")
	}
	if rollbackCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("rollback should have --no-proxy flag")
	}
}

func TestRollbackNoProxyError(t *testing.T) {
	err := noProxyRollbackError("myapp")
	msg := err.Error()
	for _, want := range []string{
		"rollback is not supported in no-proxy mode",
		"git checkout",
		"conoha app deploy --no-proxy",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %s", want, msg)
		}
	}
}
