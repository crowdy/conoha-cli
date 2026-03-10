package model

import (
	"encoding/json"
	"testing"
)

func TestServerUnmarshalFlavorRef(t *testing.T) {
	jsonData := `{
		"id": "srv-1",
		"name": "test-server",
		"status": "ACTIVE",
		"flavor": {"id": "flavor-abc"},
		"image_id": "img-1",
		"created": "2025-10-18T01:52:32Z",
		"updated": "2025-10-18T02:00:00Z"
	}`

	var s Server
	if err := json.Unmarshal([]byte(jsonData), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Flavor.ID != "flavor-abc" {
		t.Errorf("got Flavor.ID %q, want %q", s.Flavor.ID, "flavor-abc")
	}
	if s.ID != "srv-1" {
		t.Errorf("got ID %q, want %q", s.ID, "srv-1")
	}
}

func TestRemoteConsoleResponseUnmarshal(t *testing.T) {
	jsonData := `{
		"remote_console": {
			"protocol": "vnc",
			"type": "novnc",
			"url": "https://console.example.com/vnc?token=abc123"
		}
	}`

	var resp RemoteConsoleResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RemoteConsole.Protocol != "vnc" {
		t.Errorf("got Protocol %q, want %q", resp.RemoteConsole.Protocol, "vnc")
	}
	if resp.RemoteConsole.Type != "novnc" {
		t.Errorf("got Type %q, want %q", resp.RemoteConsole.Type, "novnc")
	}
	if resp.RemoteConsole.URL != "https://console.example.com/vnc?token=abc123" {
		t.Errorf("got URL %q", resp.RemoteConsole.URL)
	}
}

func TestVolumeUnmarshalFlexTime(t *testing.T) {
	jsonData := `{
		"id": "vol-1",
		"name": "test-vol",
		"status": "available",
		"size": 100,
		"created_at": "2025-10-18T01:52:32.000000"
	}`

	var v Volume
	if err := json.Unmarshal([]byte(jsonData), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if v.CreatedAt.Year() != 2025 || v.CreatedAt.Month() != 10 || v.CreatedAt.Day() != 18 {
		t.Errorf("unexpected date: %v", v.CreatedAt.Time)
	}
}
