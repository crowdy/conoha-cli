package app

import (
	"testing"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

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
