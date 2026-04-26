package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// Real ConoHa /v1/domains list endpoint returns identifiers as `uuid`.
// Before #170 the struct only declared `json:"id"`, so the entire ID slot
// came back empty and the documented `dns record add --domain-id <id>`
// flow was unusable. Lock in the wire-shape contract here.
func TestDomain_UnmarshalAcceptsUUID(t *testing.T) {
	in := []byte(`{
		"uuid": "6bb9771f-dd23-451b-ad9a-1b67298b85b2",
		"name": "example.com.",
		"email": "admin@example.com",
		"ttl": 3600,
		"serial": 1234567890,
		"status": "ACTIVE"
	}`)
	var d Domain
	if err := json.Unmarshal(in, &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.ID != "6bb9771f-dd23-451b-ad9a-1b67298b85b2" {
		t.Errorf("ID = %q, want UUID populated from `uuid` field", d.ID)
	}
	if d.Name != "example.com." {
		t.Errorf("Name = %q", d.Name)
	}
}

// The legacy field name `id` must keep working — both because mock API
// servers in unit tests use it and because some endpoints (e.g. the
// /v1/domains/<id> GET) may still return it. `id` takes precedence over
// `uuid` when both are present so a mixed-mode response decodes
// deterministically.
func TestDomain_UnmarshalPrefersIDOverUUID(t *testing.T) {
	in := []byte(`{"id": "explicit-id", "uuid": "fallback-uuid", "name": "example.com."}`)
	var d Domain
	if err := json.Unmarshal(in, &d); err != nil {
		t.Fatal(err)
	}
	if d.ID != "explicit-id" {
		t.Errorf("ID = %q, want explicit-id (id beats uuid when both present)", d.ID)
	}
}

// Marshalling preserves the legacy `id` field name so existing JSON
// consumers and table renderers don't see a shape change.
func TestDomain_MarshalKeepsIDFieldName(t *testing.T) {
	d := Domain{ID: "domain-1", Name: "example.com.", TTL: 3600}
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{`"id":"domain-1"`, `"name":"example.com."`} {
		if !strings.Contains(got, want) {
			t.Errorf("marshal missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, `"uuid":`) {
		t.Errorf("marshal must not emit `uuid` (legacy `id` is the public field): %s", got)
	}
}

func TestRecord_UnmarshalAcceptsUUID(t *testing.T) {
	in := []byte(`{
		"uuid": "d876cf43-4635-429a-8a35-2e9f5ec4c2a8",
		"name": "rc.example.com.",
		"type": "A",
		"data": "203.0.113.10",
		"ttl": 60
	}`)
	var r Record
	if err := json.Unmarshal(in, &r); err != nil {
		t.Fatal(err)
	}
	if r.ID != "d876cf43-4635-429a-8a35-2e9f5ec4c2a8" {
		t.Errorf("ID = %q, want UUID populated", r.ID)
	}
	if r.Type != "A" || r.TTL != 60 {
		t.Errorf("decoded fields wrong: %+v", r)
	}
}
