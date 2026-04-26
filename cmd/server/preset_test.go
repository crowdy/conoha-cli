package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/api"
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

func TestValidatePresetSecurityGroups_AllPresent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-groups") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
				{"id": "sg-3", "name": "IPv4v6-Web"},
				{"id": "sg-4", "name": "IPv4v6-ICMP"},
				{"id": "sg-5", "name": "extra"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	networkAPI := api.NewNetworkAPI(client)

	want := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	if err := validatePresetSecurityGroups(networkAPI, want); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidatePresetSecurityGroups_Missing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	networkAPI := api.NewNetworkAPI(client)

	want := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	err := validatePresetSecurityGroups(networkAPI, want)
	if err == nil {
		t.Fatal("expected error for missing SGs, got nil")
	}
	msg := err.Error()
	for _, missing := range []string{"IPv4v6-Web", "IPv4v6-ICMP"} {
		if !strings.Contains(msg, missing) {
			t.Errorf("error %q does not mention missing SG %q", msg, missing)
		}
	}
	for _, present := range []string{"default", "IPv4v6-SSH"} {
		if !strings.Contains(msg, present) {
			t.Errorf("error %q does not list actual SG %q (operator needs the full picture)", msg, present)
		}
	}
}

func TestValidatePresetSecurityGroups_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	networkAPI := api.NewNetworkAPI(client)

	err := validatePresetSecurityGroups(networkAPI, []string{"default"})
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}
