package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Network ---

func TestNetworkListNetworks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/networks") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"networks": []map[string]any{
				{"id": "net-1", "name": "my-network", "status": "ACTIVE"},
				{"id": "net-2", "name": "other-network", "status": "DOWN"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	networks, err := api.ListNetworks()
	if err != nil {
		t.Fatalf("ListNetworks() error: %v", err)
	}
	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(networks))
	}
	if networks[0].ID != "net-1" {
		t.Errorf("expected ID 'net-1', got %q", networks[0].ID)
	}
	if networks[0].Name != "my-network" {
		t.Errorf("expected name 'my-network', got %q", networks[0].Name)
	}
}

func TestNetworkCreateNetwork(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/networks") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		network, ok := body["network"].(map[string]any)
		if !ok {
			t.Errorf("expected 'network' key in body")
		} else if network["name"] != "new-network" {
			t.Errorf("expected name 'new-network', got %v", network["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"network": map[string]any{
				"id":     "net-new-1",
				"name":   "new-network",
				"status": "ACTIVE",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	network, err := api.CreateNetwork("new-network")
	if err != nil {
		t.Fatalf("CreateNetwork() error: %v", err)
	}
	if network.ID != "net-new-1" {
		t.Errorf("expected ID 'net-new-1', got %q", network.ID)
	}
	if network.Name != "new-network" {
		t.Errorf("expected name 'new-network', got %q", network.Name)
	}
}

func TestNetworkDeleteNetwork(t *testing.T) {
	const netID = "net-del-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/networks/"+netID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.DeleteNetwork(netID); err != nil {
		t.Fatalf("DeleteNetwork() error: %v", err)
	}
}

// --- Subnet ---

func TestSubnetListSubnets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/subnets") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"subnets": []map[string]any{
				{
					"id":         "subnet-1",
					"name":       "my-subnet",
					"network_id": "net-1",
					"cidr":       "192.168.0.0/24",
					"ip_version": 4,
				},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	subnets, err := api.ListSubnets()
	if err != nil {
		t.Fatalf("ListSubnets() error: %v", err)
	}
	if len(subnets) != 1 {
		t.Fatalf("expected 1 subnet, got %d", len(subnets))
	}
	if subnets[0].ID != "subnet-1" {
		t.Errorf("expected ID 'subnet-1', got %q", subnets[0].ID)
	}
	if subnets[0].CIDR != "192.168.0.0/24" {
		t.Errorf("expected CIDR '192.168.0.0/24', got %q", subnets[0].CIDR)
	}
}

func TestSubnetCreateSubnet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/subnets") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		subnet, ok := body["subnet"].(map[string]any)
		if !ok {
			t.Errorf("expected 'subnet' key in body")
		} else {
			if subnet["network_id"] != "net-abc" {
				t.Errorf("expected network_id 'net-abc', got %v", subnet["network_id"])
			}
			if subnet["cidr"] != "10.0.0.0/24" {
				t.Errorf("expected cidr '10.0.0.0/24', got %v", subnet["cidr"])
			}
			if subnet["name"] != "my-subnet" {
				t.Errorf("expected name 'my-subnet', got %v", subnet["name"])
			}
			if subnet["ip_version"].(float64) != 4 {
				t.Errorf("expected ip_version 4, got %v", subnet["ip_version"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"subnet": map[string]any{
				"id":         "subnet-new-1",
				"name":       "my-subnet",
				"network_id": "net-abc",
				"cidr":       "10.0.0.0/24",
				"ip_version": 4,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	subnet, err := api.CreateSubnet("net-abc", "10.0.0.0/24", "my-subnet", 4)
	if err != nil {
		t.Fatalf("CreateSubnet() error: %v", err)
	}
	if subnet.ID != "subnet-new-1" {
		t.Errorf("expected ID 'subnet-new-1', got %q", subnet.ID)
	}
	if subnet.NetworkID != "net-abc" {
		t.Errorf("expected NetworkID 'net-abc', got %q", subnet.NetworkID)
	}
}

func TestSubnetDeleteSubnet(t *testing.T) {
	const subnetID = "subnet-del-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/subnets/"+subnetID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.DeleteSubnet(subnetID); err != nil {
		t.Fatalf("DeleteSubnet() error: %v", err)
	}
}

// --- Port ---

func TestPortListPorts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/ports") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ports": []map[string]any{
				{
					"id":         "port-1",
					"name":       "my-port",
					"network_id": "net-1",
					"status":     "ACTIVE",
				},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	ports, err := api.ListPorts()
	if err != nil {
		t.Fatalf("ListPorts() error: %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(ports))
	}
	if ports[0].ID != "port-1" {
		t.Errorf("expected ID 'port-1', got %q", ports[0].ID)
	}
	if ports[0].Name != "my-port" {
		t.Errorf("expected name 'my-port', got %q", ports[0].Name)
	}
}

func TestPortCreatePort(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/ports") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		port, ok := body["port"].(map[string]any)
		if !ok {
			t.Errorf("expected 'port' key in body")
		} else {
			if port["network_id"] != "net-xyz" {
				t.Errorf("expected network_id 'net-xyz', got %v", port["network_id"])
			}
			if port["name"] != "new-port" {
				t.Errorf("expected name 'new-port', got %v", port["name"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"port": map[string]any{
				"id":         "port-new-1",
				"name":       "new-port",
				"network_id": "net-xyz",
				"status":     "DOWN",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	port, err := api.CreatePort("net-xyz", "new-port")
	if err != nil {
		t.Fatalf("CreatePort() error: %v", err)
	}
	if port.ID != "port-new-1" {
		t.Errorf("expected ID 'port-new-1', got %q", port.ID)
	}
	if port.NetworkID != "net-xyz" {
		t.Errorf("expected NetworkID 'net-xyz', got %q", port.NetworkID)
	}
}

func TestPortUpdatePort(t *testing.T) {
	const portID = "port-upd-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/ports/"+portID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		port, ok := body["port"].(map[string]any)
		if !ok {
			t.Errorf("expected 'port' key in body")
		} else if port["name"] != "updated-port" {
			t.Errorf("expected name 'updated-port', got %v", port["name"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.UpdatePort(portID, map[string]any{"name": "updated-port"}); err != nil {
		t.Fatalf("UpdatePort() error: %v", err)
	}
}

func TestPortDeletePort(t *testing.T) {
	const portID = "port-del-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/ports/"+portID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.DeletePort(portID); err != nil {
		t.Fatalf("DeletePort() error: %v", err)
	}
}

func TestPortListPortsByDevice(t *testing.T) {
	const deviceID = "server-device-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/ports") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("device_id") != deviceID {
			t.Errorf("expected device_id %q, got %q", deviceID, r.URL.Query().Get("device_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ports": []map[string]any{
				{
					"id":        "port-dev-1",
					"name":      "device-port",
					"device_id": deviceID,
					"status":    "ACTIVE",
				},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	ports, err := api.ListPortsByDevice(deviceID)
	if err != nil {
		t.Fatalf("ListPortsByDevice() error: %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(ports))
	}
	if ports[0].ID != "port-dev-1" {
		t.Errorf("expected ID 'port-dev-1', got %q", ports[0].ID)
	}
	if ports[0].DeviceID != deviceID {
		t.Errorf("expected DeviceID %q, got %q", deviceID, ports[0].DeviceID)
	}
}

// --- SecurityGroup ---

func TestSecurityGroupListSecurityGroups(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-groups") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default", "description": "Default security group"},
				{"id": "sg-2", "name": "web", "description": "Web security group"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	sgs, err := api.ListSecurityGroups()
	if err != nil {
		t.Fatalf("ListSecurityGroups() error: %v", err)
	}
	if len(sgs) != 2 {
		t.Fatalf("expected 2 security groups, got %d", len(sgs))
	}
	if sgs[0].ID != "sg-1" {
		t.Errorf("expected ID 'sg-1', got %q", sgs[0].ID)
	}
	if sgs[0].Name != "default" {
		t.Errorf("expected name 'default', got %q", sgs[0].Name)
	}
}

func TestSecurityGroupGetSecurityGroup(t *testing.T) {
	const sgID = "sg-abc-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-groups/"+sgID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"security_group": map[string]any{
				"id":          sgID,
				"name":        "my-sg",
				"description": "My security group",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	sg, err := api.GetSecurityGroup(sgID)
	if err != nil {
		t.Fatalf("GetSecurityGroup() error: %v", err)
	}
	if sg.ID != sgID {
		t.Errorf("expected ID %q, got %q", sgID, sg.ID)
	}
	if sg.Name != "my-sg" {
		t.Errorf("expected name 'my-sg', got %q", sg.Name)
	}
	if sg.Description != "My security group" {
		t.Errorf("expected description 'My security group', got %q", sg.Description)
	}
}

func TestSecurityGroupCreateSecurityGroup(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-groups") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		sg, ok := body["security_group"].(map[string]any)
		if !ok {
			t.Errorf("expected 'security_group' key in body")
		} else {
			if sg["name"] != "new-sg" {
				t.Errorf("expected name 'new-sg', got %v", sg["name"])
			}
			if sg["description"] != "New security group" {
				t.Errorf("expected description 'New security group', got %v", sg["description"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"security_group": map[string]any{
				"id":          "sg-new-1",
				"name":        "new-sg",
				"description": "New security group",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	sg, err := api.CreateSecurityGroup("new-sg", "New security group")
	if err != nil {
		t.Fatalf("CreateSecurityGroup() error: %v", err)
	}
	if sg.ID != "sg-new-1" {
		t.Errorf("expected ID 'sg-new-1', got %q", sg.ID)
	}
	if sg.Name != "new-sg" {
		t.Errorf("expected name 'new-sg', got %q", sg.Name)
	}
}

func TestSecurityGroupUpdateSecurityGroup(t *testing.T) {
	const sgID = "sg-upd-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-groups/"+sgID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		sg, ok := body["security_group"].(map[string]any)
		if !ok {
			t.Errorf("expected 'security_group' key in body")
		} else if sg["name"] != "updated-sg" {
			t.Errorf("expected name 'updated-sg', got %v", sg["name"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.UpdateSecurityGroup(sgID, map[string]any{"name": "updated-sg"}); err != nil {
		t.Fatalf("UpdateSecurityGroup() error: %v", err)
	}
}

func TestSecurityGroupDeleteSecurityGroup(t *testing.T) {
	const sgID = "sg-del-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-groups/"+sgID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.DeleteSecurityGroup(sgID); err != nil {
		t.Fatalf("DeleteSecurityGroup() error: %v", err)
	}
}

// --- SecurityGroupRule ---

func TestSecurityGroupRuleListSecurityGroupRules(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-group-rules") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"security_group_rules": []map[string]any{
				{
					"id":        "sgr-1",
					"direction": "ingress",
					"protocol":  "tcp",
					"ethertype": "IPv4",
				},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	rules, err := api.ListSecurityGroupRules()
	if err != nil {
		t.Fatalf("ListSecurityGroupRules() error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "sgr-1" {
		t.Errorf("expected ID 'sgr-1', got %q", rules[0].ID)
	}
	if rules[0].Direction != "ingress" {
		t.Errorf("expected direction 'ingress', got %q", rules[0].Direction)
	}
}

func TestSecurityGroupRuleCreateSecurityGroupRule(t *testing.T) {
	t.Run("with protocol and ports and remoteIP", func(t *testing.T) {
		portMin := 80
		portMax := 8080
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/v2.0/security-group-rules") {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			rule, ok := body["security_group_rule"].(map[string]any)
			if !ok {
				t.Errorf("expected 'security_group_rule' key in body")
			} else {
				if rule["security_group_id"] != "sg-abc" {
					t.Errorf("expected sg id 'sg-abc', got %v", rule["security_group_id"])
				}
				if rule["direction"] != "ingress" {
					t.Errorf("expected direction 'ingress', got %v", rule["direction"])
				}
				if rule["protocol"] != "tcp" {
					t.Errorf("expected protocol 'tcp', got %v", rule["protocol"])
				}
				if rule["ethertype"] != "IPv4" {
					t.Errorf("expected ethertype 'IPv4', got %v", rule["ethertype"])
				}
				if rule["port_range_min"].(float64) != 80 {
					t.Errorf("expected port_range_min 80, got %v", rule["port_range_min"])
				}
				if rule["port_range_max"].(float64) != 8080 {
					t.Errorf("expected port_range_max 8080, got %v", rule["port_range_max"])
				}
				if rule["remote_ip_prefix"] != "10.0.0.0/8" {
					t.Errorf("expected remote_ip_prefix '10.0.0.0/8', got %v", rule["remote_ip_prefix"])
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"security_group_rule": map[string]any{
					"id":               "sgr-new-1",
					"direction":        "ingress",
					"protocol":         "tcp",
					"ethertype":        "IPv4",
					"port_range_min":   80,
					"port_range_max":   8080,
					"remote_ip_prefix": "10.0.0.0/8",
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewNetworkAPI(newTestClient(ts))
		rule, err := api.CreateSecurityGroupRule("sg-abc", "ingress", "tcp", "IPv4", &portMin, &portMax, "10.0.0.0/8")
		if err != nil {
			t.Fatalf("CreateSecurityGroupRule() error: %v", err)
		}
		if rule.ID != "sgr-new-1" {
			t.Errorf("expected ID 'sgr-new-1', got %q", rule.ID)
		}
		if rule.Protocol != "tcp" {
			t.Errorf("expected protocol 'tcp', got %q", rule.Protocol)
		}
	})

	t.Run("without optional fields", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			rule, ok := body["security_group_rule"].(map[string]any)
			if !ok {
				t.Errorf("expected 'security_group_rule' key in body")
			} else {
				if _, hasProtocol := rule["protocol"]; hasProtocol {
					t.Errorf("expected no 'protocol' key when empty, but found it")
				}
				if _, hasPortMin := rule["port_range_min"]; hasPortMin {
					t.Errorf("expected no 'port_range_min' key when nil, but found it")
				}
				if _, hasPortMax := rule["port_range_max"]; hasPortMax {
					t.Errorf("expected no 'port_range_max' key when nil, but found it")
				}
				if _, hasRemoteIP := rule["remote_ip_prefix"]; hasRemoteIP {
					t.Errorf("expected no 'remote_ip_prefix' key when empty, but found it")
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"security_group_rule": map[string]any{
					"id":        "sgr-new-2",
					"direction": "egress",
					"ethertype": "IPv6",
				},
			})
		}))
		defer ts.Close()
		t.Setenv("CONOHA_ENDPOINT", ts.URL)

		api := NewNetworkAPI(newTestClient(ts))
		rule, err := api.CreateSecurityGroupRule("sg-xyz", "egress", "", "IPv6", nil, nil, "")
		if err != nil {
			t.Fatalf("CreateSecurityGroupRule() error: %v", err)
		}
		if rule.ID != "sgr-new-2" {
			t.Errorf("expected ID 'sgr-new-2', got %q", rule.ID)
		}
	})
}

func TestSecurityGroupRuleDeleteSecurityGroupRule(t *testing.T) {
	const ruleID = "sgr-del-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-group-rules/"+ruleID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.DeleteSecurityGroupRule(ruleID); err != nil {
		t.Fatalf("DeleteSecurityGroupRule() error: %v", err)
	}
}

// --- QoS ---

func TestQoSListQoSPolicies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/qos/policies") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"policies": []map[string]any{
				{"id": "qos-1", "name": "basic-qos"},
				{"id": "qos-2", "name": "premium-qos"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	policies, err := api.ListQoSPolicies()
	if err != nil {
		t.Fatalf("ListQoSPolicies() error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 QoS policies, got %d", len(policies))
	}
	if policies[0].ID != "qos-1" {
		t.Errorf("expected ID 'qos-1', got %q", policies[0].ID)
	}
	if policies[0].Name != "basic-qos" {
		t.Errorf("expected name 'basic-qos', got %q", policies[0].Name)
	}
}

// --- AttachPort / DetachPort ---

func TestAttachPort(t *testing.T) {
	const serverID = "srv-attach-1"
	const portID = "port-attach-1"

	// Compute server: handles POST /v2.1/servers/{id}/os-interface
	computeTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/v2.1/servers/" + serverID + "/os-interface"
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		iface, ok := body["interfaceAttachment"].(map[string]any)
		if !ok {
			t.Errorf("expected 'interfaceAttachment' key in body")
		} else if iface["port_id"] != portID {
			t.Errorf("expected port_id %q, got %v", portID, iface["port_id"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer computeTS.Close()

	// Network server: used only to construct the NetworkAPI (not called for AttachPort)
	networkTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected call to network server: %s %s", r.Method, r.URL.Path)
	}))
	defer networkTS.Close()

	t.Setenv("CONOHA_ENDPOINT", computeTS.URL)
	computeClient := newTestClient(computeTS)

	// For the network API, point to the network test server (though it won't be called)
	networkAPI := NewNetworkAPI(newTestClient(networkTS))

	if err := networkAPI.AttachPort(computeClient, serverID, portID); err != nil {
		t.Fatalf("AttachPort() error: %v", err)
	}
}

func TestDetachPort(t *testing.T) {
	const serverID = "srv-detach-1"
	const portID = "port-detach-1"

	// Compute server: handles DELETE /v2.1/servers/{id}/os-interface/{portID}
	computeTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		expectedPath := "/v2.1/servers/" + serverID + "/os-interface/" + portID
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer computeTS.Close()

	// Network server: used only to construct the NetworkAPI (not called for DetachPort)
	networkTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected call to network server: %s %s", r.Method, r.URL.Path)
	}))
	defer networkTS.Close()

	t.Setenv("CONOHA_ENDPOINT", computeTS.URL)
	computeClient := newTestClient(computeTS)

	networkAPI := NewNetworkAPI(newTestClient(networkTS))

	if err := networkAPI.DetachPort(computeClient, serverID, portID); err != nil {
		t.Fatalf("DetachPort() error: %v", err)
	}
}

// --- AddServerSecurityGroup / RemoveServerSecurityGroup ---

func TestAddServerSecurityGroup(t *testing.T) {
	const serverID = "srv-001"
	const sgID = "sg-abc"
	const sgName = "my-sg"
	const portID = "port-001"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v2.0/security-groups"):
			json.NewEncoder(w).Encode(map[string]any{
				"security_groups": []map[string]any{
					{"id": sgID, "name": sgName, "description": "", "security_group_rules": []any{}},
				},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v2.0/ports"):
			json.NewEncoder(w).Encode(map[string]any{
				"ports": []map[string]any{
					{"id": portID, "network_id": "net-1", "device_id": serverID, "status": "ACTIVE", "mac_address": "fa:16:3e:00:00:01", "security_groups": []string{"sg-existing"}},
				},
			})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/v2.0/ports/"+portID):
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			port := body["port"].(map[string]any)
			sgs := port["security_groups"].([]any)
			if len(sgs) != 2 {
				t.Errorf("expected 2 security groups, got %d", len(sgs))
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.AddServerSecurityGroup(serverID, sgName); err != nil {
		t.Fatalf("AddServerSecurityGroup() error: %v", err)
	}
}

func TestAddServerSecurityGroupAlreadyExists(t *testing.T) {
	const serverID = "srv-001"
	const sgID = "sg-abc"
	const sgName = "my-sg"
	const portID = "port-001"

	putCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v2.0/security-groups"):
			json.NewEncoder(w).Encode(map[string]any{
				"security_groups": []map[string]any{
					{"id": sgID, "name": sgName, "description": "", "security_group_rules": []any{}},
				},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v2.0/ports"):
			json.NewEncoder(w).Encode(map[string]any{
				"ports": []map[string]any{
					{"id": portID, "network_id": "net-1", "device_id": serverID, "status": "ACTIVE", "mac_address": "fa:16:3e:00:00:01", "security_groups": []string{sgID}},
				},
			})
		case r.Method == http.MethodPut:
			putCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.AddServerSecurityGroup(serverID, sgName); err != nil {
		t.Fatalf("AddServerSecurityGroup() error: %v", err)
	}
	if putCalled {
		t.Error("expected no PUT call when security group already exists on port")
	}
}

func TestRemoveServerSecurityGroup(t *testing.T) {
	const serverID = "srv-001"
	const sgID = "sg-abc"
	const sgName = "my-sg"
	const portID = "port-001"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v2.0/security-groups"):
			json.NewEncoder(w).Encode(map[string]any{
				"security_groups": []map[string]any{
					{"id": sgID, "name": sgName, "description": "", "security_group_rules": []any{}},
				},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v2.0/ports"):
			json.NewEncoder(w).Encode(map[string]any{
				"ports": []map[string]any{
					{"id": portID, "network_id": "net-1", "device_id": serverID, "status": "ACTIVE", "mac_address": "fa:16:3e:00:00:01", "security_groups": []string{"sg-other", sgID}},
				},
			})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/v2.0/ports/"+portID):
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			port := body["port"].(map[string]any)
			sgs := port["security_groups"].([]any)
			if len(sgs) != 1 {
				t.Errorf("expected 1 security group after removal, got %d", len(sgs))
			}
			if sgs[0] != "sg-other" {
				t.Errorf("expected remaining sg 'sg-other', got %v", sgs[0])
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.RemoveServerSecurityGroup(serverID, sgName); err != nil {
		t.Fatalf("RemoveServerSecurityGroup() error: %v", err)
	}
}

func TestRemoveServerSecurityGroupNotOnPort(t *testing.T) {
	const serverID = "srv-001"
	const sgID = "sg-abc"
	const sgName = "my-sg"
	const portID = "port-001"

	putCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v2.0/security-groups"):
			json.NewEncoder(w).Encode(map[string]any{
				"security_groups": []map[string]any{
					{"id": sgID, "name": sgName, "description": "", "security_group_rules": []any{}},
				},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v2.0/ports"):
			json.NewEncoder(w).Encode(map[string]any{
				"ports": []map[string]any{
					{"id": portID, "network_id": "net-1", "device_id": serverID, "status": "ACTIVE", "mac_address": "fa:16:3e:00:00:01", "security_groups": []string{"sg-other"}},
				},
			})
		case r.Method == http.MethodPut:
			putCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	if err := api.RemoveServerSecurityGroup(serverID, sgName); err != nil {
		t.Fatalf("RemoveServerSecurityGroup() error: %v", err)
	}
	if putCalled {
		t.Error("expected no PUT call when security group is not on any port")
	}
}

func TestAddServerSecurityGroupNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"security_groups": []any{}})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewNetworkAPI(newTestClient(ts))
	err := api.AddServerSecurityGroup("srv-001", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent security group")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}
