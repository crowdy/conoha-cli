package server

import (
	"strings"
	"testing"
)

func TestMatchDockerUbuntuAmd64(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"happy", "vmi-docker-25.10-ubuntu-22.04-amd64", true},
		{"happy_alt_version", "vmi-docker-26.04-ubuntu-24.04-amd64", true},
		{"missing_docker", "vmi-ubuntu-22.04-amd64", false},
		{"non_ubuntu", "vmi-docker-25.10-rocky-9-amd64", false},
		{"non_amd64", "vmi-docker-25.10-ubuntu-22.04-arm64", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchDockerUbuntuAmd64(tc.in)
			if got != tc.want {
				t.Errorf("matchDockerUbuntuAmd64(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestKnownPresetList(t *testing.T) {
	got := knownPresetList()
	if !strings.Contains(got, "proxy") {
		t.Errorf("knownPresetList() = %q, want it to contain %q", got, "proxy")
	}
}

func TestPresetRegistry_HasProxy(t *testing.T) {
	spec, ok := presets["proxy"]
	if !ok {
		t.Fatal("presets[\"proxy\"] missing")
	}
	if spec.Flavor != "g2l-t-c3m2" {
		t.Errorf("proxy flavor = %q, want %q", spec.Flavor, "g2l-t-c3m2")
	}
	wantSGs := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	if len(spec.SecurityGroups) != len(wantSGs) {
		t.Fatalf("proxy SGs len = %d, want %d", len(spec.SecurityGroups), len(wantSGs))
	}
	for i, n := range wantSGs {
		if spec.SecurityGroups[i] != n {
			t.Errorf("proxy SG[%d] = %q, want %q", i, spec.SecurityGroups[i], n)
		}
	}
	if spec.ImageMatch == nil {
		t.Error("proxy ImageMatch is nil")
	}
}
