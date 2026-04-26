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

func TestResolvePresetImage_PicksLatest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v2/images") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-old", "name": "vmi-docker-25.04-ubuntu-22.04-amd64", "status": "active"},
				{"id": "img-new", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
				{"id": "img-other", "name": "vmi-docker-25.10-rocky-9-amd64", "status": "active"},
				{"id": "img-deactivated", "name": "vmi-docker-26.04-ubuntu-24.04-amd64", "status": "deactivated"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)

	id, err := resolvePresetImage(imageAPI, matchDockerUbuntuAmd64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "img-new" {
		t.Errorf("resolved image = %q, want %q (newer name should win, deactivated should not)", id, "img-new")
	}
}

func TestResolvePresetImage_NoMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-1", "name": "vmi-ubuntu-24.04-amd64", "status": "active"},
				{"id": "img-2", "name": "vmi-docker-25.10-rocky-9-amd64", "status": "active"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)

	_, err := resolvePresetImage(imageAPI, matchDockerUbuntuAmd64)
	if err == nil {
		t.Fatal("expected error for no-match list, got nil")
	}
	if !strings.Contains(err.Error(), "no image") {
		t.Errorf("error %q does not mention 'no image'", err.Error())
	}
}

func TestResolvePresetImage_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)

	_, err := resolvePresetImage(imageAPI, matchDockerUbuntuAmd64)
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}

func TestResolvePreset_AllDefaults(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/images", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-latest", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
			},
		})
	})
	mux.HandleFunc("GET /v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
				{"id": "sg-3", "name": "IPv4v6-Web"},
				{"id": "sg-4", "name": "IPv4v6-ICMP"},
			},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)
	networkAPI := api.NewNetworkAPI(client)

	flavor, image, sgs, err := resolvePreset("proxy", "", "", nil, imageAPI, networkAPI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != "g2l-t-c3m2" {
		t.Errorf("flavor = %q, want %q", flavor, "g2l-t-c3m2")
	}
	if image != "img-latest" {
		t.Errorf("image = %q, want %q", image, "img-latest")
	}
	wantSGs := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	if len(sgs) != len(wantSGs) {
		t.Fatalf("sgs len = %d, want %d", len(sgs), len(wantSGs))
	}
	for i, n := range wantSGs {
		if sgs[i] != n {
			t.Errorf("sgs[%d] = %q, want %q", i, sgs[i], n)
		}
	}
}

func TestResolvePreset_ExplicitFlavorWins(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/images", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-latest", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
			},
		})
	})
	mux.HandleFunc("GET /v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
				{"id": "sg-3", "name": "IPv4v6-Web"},
				{"id": "sg-4", "name": "IPv4v6-ICMP"},
			},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	flavor, image, _, err := resolvePreset("proxy", "g2l-t-c4m4", "", nil,
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != "g2l-t-c4m4" {
		t.Errorf("flavor = %q, want explicit %q", flavor, "g2l-t-c4m4")
	}
	if image != "img-latest" {
		t.Errorf("image = %q, want preset-resolved %q", image, "img-latest")
	}
}

func TestResolvePreset_ExplicitSGReplaces(t *testing.T) {
	// When user passes their own --security-group, the preset's SG list is
	// REPLACED, not appended. Verify the API is not even called for SG
	// validation (would be wasted work and could give a misleading error).
	mux := http.NewServeMux()
	sgCalls := 0
	mux.HandleFunc("GET /v2/images", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-latest", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
			},
		})
	})
	mux.HandleFunc("GET /v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		sgCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	_, _, sgs, err := resolvePreset("proxy", "", "", []string{"custom"},
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sgs) != 1 || sgs[0] != "custom" {
		t.Errorf("sgs = %v, want [custom]", sgs)
	}
	if sgCalls != 0 {
		t.Errorf("ListSecurityGroups was called %d times; should not be called when user passed explicit SGs", sgCalls)
	}
}

func TestResolvePreset_UnknownName(t *testing.T) {
	client := &api.Client{Token: "fake-token", TenantID: "tenant-1"}
	_, _, _, err := resolvePreset("nonexistent", "", "", nil,
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err == nil {
		t.Fatal("expected error for unknown preset, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "nonexistent") {
		t.Errorf("error %q does not mention bad name", msg)
	}
	if !strings.Contains(msg, "proxy") {
		t.Errorf("error %q does not list known preset 'proxy'", msg)
	}
}

func TestResolvePreset_EmptyName_NoOp(t *testing.T) {
	// Empty preset name means user did not pass --for; resolvePreset must
	// return its inputs unchanged and make zero API calls.
	client := &api.Client{Token: "fake-token", TenantID: "tenant-1"}
	flavor, image, sgs, err := resolvePreset("", "fl-1", "im-1", []string{"sg-1"},
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != "fl-1" || image != "im-1" || len(sgs) != 1 || sgs[0] != "sg-1" {
		t.Errorf("expected pass-through, got flavor=%q image=%q sgs=%v", flavor, image, sgs)
	}
}

func TestCreateCmd_ForFlagRegistered(t *testing.T) {
	f := createCmd.Flags().Lookup("for")
	if f == nil {
		t.Fatal("create command missing --for flag")
	}
	if f.DefValue != "" {
		t.Errorf("--for default = %q, want empty", f.DefValue)
	}
}
