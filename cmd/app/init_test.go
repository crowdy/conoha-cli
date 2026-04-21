package app

import (
	"testing"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func TestInitCmd_HasModeFlags(t *testing.T) {
	if initCmd.Flags().Lookup("proxy") == nil {
		t.Error("init should have --proxy flag")
	}
	if initCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("init should have --no-proxy flag")
	}
}

func TestInitCmd_ModeFlagsMutuallyExclusive(t *testing.T) {
	// ParseFlags alone does not validate mutual exclusion in cobra;
	// ValidateFlagGroups is the correct API for that check.
	if err := initCmd.ParseFlags([]string{"--proxy", "--no-proxy"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := initCmd.ValidateFlagGroups(); err == nil {
		t.Error("--proxy and --no-proxy should be mutually exclusive")
	}
	// Reset flags for subsequent tests.
	_ = initCmd.Flags().Set("proxy", "false")
	_ = initCmd.Flags().Set("no-proxy", "false")
}

func TestMapHealth_Nil(t *testing.T) {
	if got := mapHealth(nil); got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestMapHealth_AllFields(t *testing.T) {
	in := &config.HealthSpec{
		Path: "/up", IntervalMs: 1000, TimeoutMs: 500,
		HealthyThreshold: 2, UnhealthyThreshold: 5,
	}
	want := &proxypkg.HealthPolicy{
		Path: "/up", IntervalMs: 1000, TimeoutMs: 500,
		HealthyThreshold: 2, UnhealthyThreshold: 5,
	}
	got := mapHealth(in)
	if *got != *want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
