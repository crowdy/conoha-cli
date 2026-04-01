package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/model"
)

func TestListServers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/detail") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"servers": []map[string]any{
				{"id": "server-1", "name": "test-server", "status": "ACTIVE"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	servers, err := api.ListServers()
	if err != nil {
		t.Fatalf("ListServers() error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].ID != "server-1" {
		t.Errorf("expected server ID 'server-1', got %q", servers[0].ID)
	}
	if servers[0].Name != "test-server" {
		t.Errorf("expected server name 'test-server', got %q", servers[0].Name)
	}
}

func TestGetServer(t *testing.T) {
	const serverID = "abc-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"server": map[string]any{
				"id":     serverID,
				"name":   "my-server",
				"status": "ACTIVE",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	server, err := api.GetServer(serverID)
	if err != nil {
		t.Fatalf("GetServer() error: %v", err)
	}
	if server.ID != serverID {
		t.Errorf("expected server ID %q, got %q", serverID, server.ID)
	}
	if server.Name != "my-server" {
		t.Errorf("expected server name 'my-server', got %q", server.Name)
	}
}

func TestFindServer(t *testing.T) {
	t.Run("by UUID", func(t *testing.T) {
		const uuid = "12345678-1234-1234-1234-123456789abc"
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"server": map[string]any{
					"id":   uuid,
					"name": "uuid-server",
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		server, err := api.FindServer(uuid)
		if err != nil {
			t.Fatalf("FindServer() error: %v", err)
		}
		if server.ID != uuid {
			t.Errorf("expected ID %q, got %q", uuid, server.ID)
		}
	})

	t.Run("by name", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// FindServer by name calls ListServers (servers/detail)
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]any{
					{"id": "srv-456", "name": "named-server", "status": "ACTIVE"},
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		server, err := api.FindServer("named-server")
		if err != nil {
			t.Fatalf("FindServer() error: %v", err)
		}
		if server.ID != "srv-456" {
			t.Errorf("expected ID 'srv-456', got %q", server.ID)
		}
	})

	t.Run("by nametag", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]any{
					{
						"id":     "srv-789",
						"name":   "random-vm-name",
						"status": "ACTIVE",
						"metadata": map[string]string{
							"instance_name_tag": "my-web-server",
						},
					},
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		server, err := api.FindServer("my-web-server")
		if err != nil {
			t.Fatalf("FindServer() error: %v", err)
		}
		if server.ID != "srv-789" {
			t.Errorf("expected ID 'srv-789', got %q", server.ID)
		}
	})

	t.Run("name wins over nametag", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]any{
					{
						"id":     "srv-name",
						"name":   "foo",
						"status": "ACTIVE",
					},
					{
						"id":     "srv-tag",
						"name":   "other-vm",
						"status": "ACTIVE",
						"metadata": map[string]string{
							"instance_name_tag": "foo",
						},
					},
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		server, err := api.FindServer("foo")
		if err != nil {
			t.Fatalf("FindServer() error: %v", err)
		}
		if server.ID != "srv-name" {
			t.Errorf("expected name match 'srv-name', got %q (nametag should not win)", server.ID)
		}
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]any{
					{
						"id":     "srv-dup-1",
						"name":   "dup-name",
						"status": "ACTIVE",
					},
					{
						"id":     "srv-dup-2",
						"name":   "dup-name",
						"status": "ACTIVE",
					},
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		_, err := api.FindServer("dup-name")
		if err == nil {
			t.Fatal("expected error for duplicate name, got nil")
		}
		if !strings.Contains(err.Error(), "multiple servers found with name") {
			t.Errorf("expected 'multiple servers found with name' in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "srv-dup-1") || !strings.Contains(err.Error(), "srv-dup-2") {
			t.Errorf("expected server IDs in error, got: %v", err)
		}
	})

	t.Run("duplicate nametag returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]any{
					{
						"id":     "srv-dup-1",
						"name":   "vm-a",
						"status": "ACTIVE",
						"metadata": map[string]string{
							"instance_name_tag": "dup",
						},
					},
					{
						"id":     "srv-dup-2",
						"name":   "vm-b",
						"status": "ACTIVE",
						"metadata": map[string]string{
							"instance_name_tag": "dup",
						},
					},
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		_, err := api.FindServer("dup")
		if err == nil {
			t.Fatal("expected error for duplicate nametag, got nil")
		}
		if !strings.Contains(err.Error(), "multiple servers") {
			t.Errorf("expected 'multiple servers' in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "srv-dup-1") || !strings.Contains(err.Error(), "srv-dup-2") {
			t.Errorf("expected server IDs in error, got: %v", err)
		}
	})
}

func TestCreateServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		serverMap, ok := body["server"].(map[string]any)
		if !ok {
			t.Errorf("expected 'server' key in body")
		} else if serverMap["name"] != "new-server" {
			t.Errorf("expected name 'new-server', got %v", serverMap["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"server": map[string]any{
				"id":   "new-srv-id",
				"name": "new-server",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	req := &model.ServerCreateRequest{}
	req.Server.Name = "new-server"
	req.Server.FlavorRef = "g2l-c2m4d100"
	server, err := api.CreateServer(req)
	if err != nil {
		t.Fatalf("CreateServer() error: %v", err)
	}
	if server.ID != "new-srv-id" {
		t.Errorf("expected ID 'new-srv-id', got %q", server.ID)
	}
}

func TestDeleteServer(t *testing.T) {
	const serverID = "del-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.DeleteServer(serverID); err != nil {
		t.Fatalf("DeleteServer() error: %v", err)
	}
}

func TestServerAction(t *testing.T) {
	const serverID = "action-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/action") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.ServerAction(serverID, map[string]any{"os-start": nil}); err != nil {
		t.Fatalf("ServerAction() error: %v", err)
	}
}

func TestStartServer(t *testing.T) {
	const serverID = "start-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["os-start"]; !ok {
			t.Errorf("expected 'os-start' key in body, got %v", body)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.StartServer(serverID); err != nil {
		t.Fatalf("StartServer() error: %v", err)
	}
}

func TestStopServer(t *testing.T) {
	const serverID = "stop-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["os-stop"]; !ok {
			t.Errorf("expected 'os-stop' key in body, got %v", body)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.StopServer(serverID); err != nil {
		t.Fatalf("StopServer() error: %v", err)
	}
}

func TestRebootServer(t *testing.T) {
	const serverID = "reboot-srv-id"

	t.Run("soft reboot", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			reboot, ok := body["reboot"].(map[string]any)
			if !ok {
				t.Errorf("expected 'reboot' key in body")
			} else if reboot["type"] != "SOFT" {
				t.Errorf("expected type 'SOFT', got %v", reboot["type"])
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		if err := api.RebootServer(serverID, false); err != nil {
			t.Fatalf("RebootServer(soft) error: %v", err)
		}
	})

	t.Run("hard reboot", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			reboot, ok := body["reboot"].(map[string]any)
			if !ok {
				t.Errorf("expected 'reboot' key in body")
			} else if reboot["type"] != "HARD" {
				t.Errorf("expected type 'HARD', got %v", reboot["type"])
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewComputeAPI(newTestClient(ts))
		if err := api.RebootServer(serverID, true); err != nil {
			t.Fatalf("RebootServer(hard) error: %v", err)
		}
	})
}

func TestResizeServer(t *testing.T) {
	const serverID = "resize-srv-id"
	const flavorID = "g2l-c4m8d100"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		resize, ok := body["resize"].(map[string]any)
		if !ok {
			t.Errorf("expected 'resize' key in body")
		} else if resize["flavorRef"] != flavorID {
			t.Errorf("expected flavorRef %q, got %v", flavorID, resize["flavorRef"])
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.ResizeServer(serverID, flavorID); err != nil {
		t.Fatalf("ResizeServer() error: %v", err)
	}
}

func TestRebuildServer(t *testing.T) {
	const serverID = "rebuild-srv-id"
	const imageID = "image-abc-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		rebuild, ok := body["rebuild"].(map[string]any)
		if !ok {
			t.Errorf("expected 'rebuild' key in body")
		} else if rebuild["imageRef"] != imageID {
			t.Errorf("expected imageRef %q, got %v", imageID, rebuild["imageRef"])
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.RebuildServer(serverID, imageID); err != nil {
		t.Fatalf("RebuildServer() error: %v", err)
	}
}

func TestGetConsole(t *testing.T) {
	const serverID = "console-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/remote-consoles") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		console, ok := body["remote_console"].(map[string]any)
		if !ok {
			t.Errorf("expected 'remote_console' key in body")
		} else {
			if console["protocol"] != "vnc" {
				t.Errorf("expected protocol 'vnc', got %v", console["protocol"])
			}
			if console["type"] != "novnc" {
				t.Errorf("expected type 'novnc', got %v", console["type"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"remote_console": map[string]any{
				"protocol": "vnc",
				"type":     "novnc",
				"url":      "https://console.example.com/vnc",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	console, err := api.GetConsole(serverID)
	if err != nil {
		t.Fatalf("GetConsole() error: %v", err)
	}
	if console.RemoteConsole.URL != "https://console.example.com/vnc" {
		t.Errorf("expected URL 'https://console.example.com/vnc', got %q", console.RemoteConsole.URL)
	}
	if console.RemoteConsole.Protocol != "vnc" {
		t.Errorf("expected protocol 'vnc', got %q", console.RemoteConsole.Protocol)
	}
}

func TestListFlavors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/flavors/detail") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"flavors": []map[string]any{
				{"id": "flavor-1", "name": "g2l-c2m4d100", "ram": 4096, "vcpus": 2, "disk": 100},
				{"id": "flavor-2", "name": "g2l-c4m8d100", "ram": 8192, "vcpus": 4, "disk": 100},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	flavors, err := api.ListFlavors()
	if err != nil {
		t.Fatalf("ListFlavors() error: %v", err)
	}
	if len(flavors) != 2 {
		t.Fatalf("expected 2 flavors, got %d", len(flavors))
	}
	if flavors[0].ID != "flavor-1" {
		t.Errorf("expected flavor ID 'flavor-1', got %q", flavors[0].ID)
	}
}

func TestGetFlavor(t *testing.T) {
	const flavorID = "flavor-xyz"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/flavors/"+flavorID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"flavor": map[string]any{
				"id":    flavorID,
				"name":  "g2l-c2m4d100",
				"ram":   4096,
				"vcpus": 2,
				"disk":  100,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	flavor, err := api.GetFlavor(flavorID)
	if err != nil {
		t.Fatalf("GetFlavor() error: %v", err)
	}
	if flavor.ID != flavorID {
		t.Errorf("expected flavor ID %q, got %q", flavorID, flavor.ID)
	}
	if flavor.RAM != 4096 {
		t.Errorf("expected RAM 4096, got %d", flavor.RAM)
	}
}

func TestListKeypairs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/os-keypairs") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// Nested structure: keypairs -> [keypair wrapper] -> keypair
		json.NewEncoder(w).Encode(map[string]any{
			"keypairs": []map[string]any{
				{
					"keypair": map[string]any{
						"name":        "my-key",
						"fingerprint": "aa:bb:cc",
						"public_key":  "ssh-rsa AAAA...",
					},
				},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	keypairs, err := api.ListKeypairs()
	if err != nil {
		t.Fatalf("ListKeypairs() error: %v", err)
	}
	if len(keypairs) != 1 {
		t.Fatalf("expected 1 keypair, got %d", len(keypairs))
	}
	if keypairs[0].Name != "my-key" {
		t.Errorf("expected name 'my-key', got %q", keypairs[0].Name)
	}
	if keypairs[0].Fingerprint != "aa:bb:cc" {
		t.Errorf("expected fingerprint 'aa:bb:cc', got %q", keypairs[0].Fingerprint)
	}
}

func TestCreateKeypair(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/os-keypairs") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		kp, ok := body["keypair"].(map[string]any)
		if !ok {
			t.Errorf("expected 'keypair' key in body")
		} else if kp["name"] != "new-key" {
			t.Errorf("expected name 'new-key', got %v", kp["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"keypair": map[string]any{
				"name":        "new-key",
				"fingerprint": "11:22:33",
				"public_key":  "ssh-rsa BBBB...",
				"private_key": "-----BEGIN RSA PRIVATE KEY-----\n...",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	req := &model.KeypairCreateRequest{}
	req.Keypair.Name = "new-key"
	keypair, err := api.CreateKeypair(req)
	if err != nil {
		t.Fatalf("CreateKeypair() error: %v", err)
	}
	if keypair.Name != "new-key" {
		t.Errorf("expected name 'new-key', got %q", keypair.Name)
	}
	if keypair.PrivateKey == "" {
		t.Errorf("expected private key to be set")
	}
}

func TestDeleteKeypair(t *testing.T) {
	const keyName = "old-key"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/os-keypairs/"+keyName) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.DeleteKeypair(keyName); err != nil {
		t.Fatalf("DeleteKeypair() error: %v", err)
	}
}

func TestListVolumeAttachments(t *testing.T) {
	const serverID = "vol-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/os-volume_attachments") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"volumeAttachments": []map[string]any{
				{
					"id":       "attach-1",
					"volumeId": "vol-111",
					"device":   "/dev/vda",
					"serverId": serverID,
				},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	attachments, err := api.ListVolumeAttachments(serverID)
	if err != nil {
		t.Fatalf("ListVolumeAttachments() error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].VolumeID != "vol-111" {
		t.Errorf("expected VolumeID 'vol-111', got %q", attachments[0].VolumeID)
	}
	if attachments[0].Device != "/dev/vda" {
		t.Errorf("expected device '/dev/vda', got %q", attachments[0].Device)
	}
}

func TestGetServerMetadata(t *testing.T) {
	const serverID = "meta-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/metadata") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"metadata": map[string]string{
				"env":  "production",
				"role": "web",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	metadata, err := api.GetServerMetadata(serverID)
	if err != nil {
		t.Fatalf("GetServerMetadata() error: %v", err)
	}
	if metadata["env"] != "production" {
		t.Errorf("expected env 'production', got %q", metadata["env"])
	}
	if metadata["role"] != "web" {
		t.Errorf("expected role 'web', got %q", metadata["role"])
	}
}

func TestAttachVolume(t *testing.T) {
	const serverID = "attach-srv-id"
	const volumeID = "attach-vol-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/os-volume_attachments") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		va, ok := body["volumeAttachment"].(map[string]any)
		if !ok {
			t.Errorf("expected 'volumeAttachment' key in body")
		} else if va["volumeId"] != volumeID {
			t.Errorf("expected volumeId %q, got %v", volumeID, va["volumeId"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.AttachVolume(serverID, volumeID); err != nil {
		t.Fatalf("AttachVolume() error: %v", err)
	}
}

func TestDetachVolume(t *testing.T) {
	const serverID = "detach-srv-id"
	const volumeID = "detach-vol-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		expectedPath := "/v2.1/servers/" + serverID + "/os-volume_attachments/" + volumeID
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.DetachVolume(serverID, volumeID); err != nil {
		t.Fatalf("DetachVolume() error: %v", err)
	}
}

func TestAddSecurityGroup(t *testing.T) {
	const serverID = "sg-add-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/action") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		sg, ok := body["addSecurityGroup"].(map[string]any)
		if !ok {
			t.Errorf("expected 'addSecurityGroup' key in body, got %v", body)
		} else if sg["name"] != "my-sg" {
			t.Errorf("expected name 'my-sg', got %v", sg["name"])
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.AddSecurityGroup(serverID, "my-sg"); err != nil {
		t.Fatalf("AddSecurityGroup() error: %v", err)
	}
}

func TestRemoveSecurityGroup(t *testing.T) {
	const serverID = "sg-rm-srv-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.1/servers/"+serverID+"/action") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		sg, ok := body["removeSecurityGroup"].(map[string]any)
		if !ok {
			t.Errorf("expected 'removeSecurityGroup' key in body, got %v", body)
		} else if sg["name"] != "old-sg" {
			t.Errorf("expected name 'old-sg', got %v", sg["name"])
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewComputeAPI(newTestClient(ts))
	if err := api.RemoveSecurityGroup(serverID, "old-sg"); err != nil {
		t.Fatalf("RemoveSecurityGroup() error: %v", err)
	}
}
