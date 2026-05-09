package server

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/crowdy/conoha-cli/internal/model"
)

// #191: server show --format json must include security_groups, not null.
// The detail struct embeds *model.Server so existing fields stay at the
// top level of the JSON object (and YAML mapping).
func TestServerShowDetail_JSONShape(t *testing.T) {
	d := serverShowDetail{
		Server:         &model.Server{ID: "srv-abc", Name: "web1", Status: "ACTIVE"},
		SecurityGroups: []string{"IPv4v6-SSH", "3000-9999"},
		Ports: []serverShowDetailPort{
			{ID: "port-1", MACAddress: "fa:16:3e:01:02:03", IPs: []string{"10.0.0.5"}},
		},
		Volumes: []serverShowDetailVolume{
			{ID: "vol-1", Device: "/dev/vda", SizeGB: 100},
		},
	}

	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	out := string(b)

	for _, want := range []string{
		`"id":"srv-abc"`,
		`"name":"web1"`,
		`"security_groups":["IPv4v6-SSH","3000-9999"]`,
		`"ports":[`,
		`"volumes":[`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected JSON to contain %q, got: %s", want, out)
		}
	}

	// Round-trip: structured consumers must be able to read security_groups.
	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	sgs, ok := parsed["security_groups"].([]any)
	if !ok || len(sgs) != 2 {
		t.Errorf("security_groups not a 2-element array: %v", parsed["security_groups"])
	}
}

func TestServerShowDetail_YAMLShape(t *testing.T) {
	d := serverShowDetail{
		Server:         &model.Server{ID: "srv-abc", Name: "web1"},
		SecurityGroups: []string{"web"},
	}
	b, err := yaml.Marshal(d)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	out := string(b)
	// embedded *model.Server should be inlined (no `server:` nesting)
	if strings.Contains(out, "Server:") || strings.Contains(out, "server:") {
		t.Errorf("expected embedded struct to be inlined, got: %s", out)
	}
	if !strings.Contains(out, "id: srv-abc") || !strings.Contains(out, "security_groups:") {
		t.Errorf("expected flattened YAML, got: %s", out)
	}
}

// When SecurityGroups is empty, omitempty must drop the field rather
// than emitting `"security_groups":null` (which would re-introduce the
// original bug shape).
func TestServerShowDetail_EmptySGsOmitted(t *testing.T) {
	d := serverShowDetail{Server: &model.Server{ID: "srv-x"}}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(b), "security_groups") {
		t.Errorf("expected security_groups to be omitted when nil, got: %s", string(b))
	}
}
