package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- LoadBalancer ---

func TestListLoadBalancers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/loadbalancers") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"loadbalancers": []map[string]any{
				{"id": "lb-1", "name": "my-lb", "provisioning_status": "ACTIVE"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	lbs, err := api.ListLoadBalancers()
	if err != nil {
		t.Fatalf("ListLoadBalancers() error: %v", err)
	}
	if len(lbs) != 1 {
		t.Fatalf("expected 1 load balancer, got %d", len(lbs))
	}
	if lbs[0].ID != "lb-1" {
		t.Errorf("expected ID 'lb-1', got %q", lbs[0].ID)
	}
	if lbs[0].Name != "my-lb" {
		t.Errorf("expected name 'my-lb', got %q", lbs[0].Name)
	}
}

func TestGetLoadBalancer(t *testing.T) {
	const lbID = "lb-abc-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/loadbalancers/"+lbID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"loadbalancer": map[string]any{
				"id":                  lbID,
				"name":                "test-lb",
				"provisioning_status": "ACTIVE",
				"vip_address":         "192.168.1.100",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	lb, err := api.GetLoadBalancer(lbID)
	if err != nil {
		t.Fatalf("GetLoadBalancer() error: %v", err)
	}
	if lb.ID != lbID {
		t.Errorf("expected ID %q, got %q", lbID, lb.ID)
	}
	if lb.VipAddress != "192.168.1.100" {
		t.Errorf("expected VipAddress '192.168.1.100', got %q", lb.VipAddress)
	}
}

func TestCreateLoadBalancer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/loadbalancers") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		lbMap, ok := body["loadbalancer"].(map[string]any)
		if !ok {
			t.Errorf("expected 'loadbalancer' key in body")
		} else {
			if lbMap["name"] != "new-lb" {
				t.Errorf("expected name 'new-lb', got %v", lbMap["name"])
			}
			if lbMap["vip_subnet_id"] != "subnet-123" {
				t.Errorf("expected vip_subnet_id 'subnet-123', got %v", lbMap["vip_subnet_id"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"loadbalancer": map[string]any{
				"id":            "lb-new",
				"name":          "new-lb",
				"vip_subnet_id": "subnet-123",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	lb, err := api.CreateLoadBalancer("new-lb", "subnet-123")
	if err != nil {
		t.Fatalf("CreateLoadBalancer() error: %v", err)
	}
	if lb.ID != "lb-new" {
		t.Errorf("expected ID 'lb-new', got %q", lb.ID)
	}
	if lb.Name != "new-lb" {
		t.Errorf("expected name 'new-lb', got %q", lb.Name)
	}
}

func TestUpdateLoadBalancer(t *testing.T) {
	const lbID = "lb-upd-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/loadbalancers/"+lbID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		lbMap, ok := body["loadbalancer"].(map[string]any)
		if !ok {
			t.Errorf("expected 'loadbalancer' key in body")
		} else if lbMap["name"] != "updated-lb" {
			t.Errorf("expected name 'updated-lb', got %v", lbMap["name"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	err := api.UpdateLoadBalancer(lbID, map[string]any{"name": "updated-lb"})
	if err != nil {
		t.Fatalf("UpdateLoadBalancer() error: %v", err)
	}
}

func TestDeleteLoadBalancer(t *testing.T) {
	const lbID = "lb-del-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/loadbalancers/"+lbID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	if err := api.DeleteLoadBalancer(lbID); err != nil {
		t.Fatalf("DeleteLoadBalancer() error: %v", err)
	}
}

// --- Listener ---

func TestListListeners(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/listeners") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"listeners": []map[string]any{
				{"id": "listener-1", "name": "http-listener", "protocol": "HTTP", "protocol_port": 80},
				{"id": "listener-2", "name": "https-listener", "protocol": "HTTPS", "protocol_port": 443},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	listeners, err := api.ListListeners()
	if err != nil {
		t.Fatalf("ListListeners() error: %v", err)
	}
	if len(listeners) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(listeners))
	}
	if listeners[0].ID != "listener-1" {
		t.Errorf("expected ID 'listener-1', got %q", listeners[0].ID)
	}
	if listeners[0].Protocol != "HTTP" {
		t.Errorf("expected protocol 'HTTP', got %q", listeners[0].Protocol)
	}
}

func TestGetListener(t *testing.T) {
	const listenerID = "listener-xyz"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/listeners/"+listenerID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"listener": map[string]any{
				"id":            listenerID,
				"name":          "my-listener",
				"protocol":      "TCP",
				"protocol_port": 8080,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	listener, err := api.GetListener(listenerID)
	if err != nil {
		t.Fatalf("GetListener() error: %v", err)
	}
	if listener.ID != listenerID {
		t.Errorf("expected ID %q, got %q", listenerID, listener.ID)
	}
	if listener.ProtocolPort != 8080 {
		t.Errorf("expected protocol_port 8080, got %d", listener.ProtocolPort)
	}
}

func TestCreateListener(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/listeners") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		listenerMap, ok := body["listener"].(map[string]any)
		if !ok {
			t.Errorf("expected 'listener' key in body")
		} else {
			if listenerMap["name"] != "new-listener" {
				t.Errorf("expected name 'new-listener', got %v", listenerMap["name"])
			}
			if listenerMap["protocol"] != "HTTP" {
				t.Errorf("expected protocol 'HTTP', got %v", listenerMap["protocol"])
			}
			if listenerMap["protocol_port"] != float64(80) {
				t.Errorf("expected protocol_port 80, got %v", listenerMap["protocol_port"])
			}
			if listenerMap["loadbalancer_id"] != "lb-123" {
				t.Errorf("expected loadbalancer_id 'lb-123', got %v", listenerMap["loadbalancer_id"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"listener": map[string]any{
				"id":              "listener-new",
				"name":            "new-listener",
				"protocol":        "HTTP",
				"protocol_port":   80,
				"loadbalancer_id": "lb-123",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	listener, err := api.CreateListener("new-listener", "HTTP", 80, "lb-123")
	if err != nil {
		t.Fatalf("CreateListener() error: %v", err)
	}
	if listener.ID != "listener-new" {
		t.Errorf("expected ID 'listener-new', got %q", listener.ID)
	}
	if listener.Name != "new-listener" {
		t.Errorf("expected name 'new-listener', got %q", listener.Name)
	}
}

func TestDeleteListener(t *testing.T) {
	const listenerID = "listener-del-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/listeners/"+listenerID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	if err := api.DeleteListener(listenerID); err != nil {
		t.Fatalf("DeleteListener() error: %v", err)
	}
}

// --- Pool ---

func TestListPools(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/pools") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"pools": []map[string]any{
				{"id": "pool-1", "name": "my-pool", "protocol": "HTTP", "lb_algorithm": "ROUND_ROBIN"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	pools, err := api.ListPools()
	if err != nil {
		t.Fatalf("ListPools() error: %v", err)
	}
	if len(pools) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(pools))
	}
	if pools[0].ID != "pool-1" {
		t.Errorf("expected ID 'pool-1', got %q", pools[0].ID)
	}
	if pools[0].LBMethod != "ROUND_ROBIN" {
		t.Errorf("expected lb_algorithm 'ROUND_ROBIN', got %q", pools[0].LBMethod)
	}
}

func TestGetPool(t *testing.T) {
	const poolID = "pool-xyz"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/pools/"+poolID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"pool": map[string]any{
				"id":           poolID,
				"name":         "test-pool",
				"protocol":     "TCP",
				"lb_algorithm": "LEAST_CONNECTIONS",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	pool, err := api.GetPool(poolID)
	if err != nil {
		t.Fatalf("GetPool() error: %v", err)
	}
	if pool.ID != poolID {
		t.Errorf("expected ID %q, got %q", poolID, pool.ID)
	}
	if pool.LBMethod != "LEAST_CONNECTIONS" {
		t.Errorf("expected lb_algorithm 'LEAST_CONNECTIONS', got %q", pool.LBMethod)
	}
}

func TestCreatePool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/pools") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		poolMap, ok := body["pool"].(map[string]any)
		if !ok {
			t.Errorf("expected 'pool' key in body")
		} else {
			if poolMap["name"] != "new-pool" {
				t.Errorf("expected name 'new-pool', got %v", poolMap["name"])
			}
			if poolMap["protocol"] != "HTTP" {
				t.Errorf("expected protocol 'HTTP', got %v", poolMap["protocol"])
			}
			if poolMap["lb_algorithm"] != "ROUND_ROBIN" {
				t.Errorf("expected lb_algorithm 'ROUND_ROBIN', got %v", poolMap["lb_algorithm"])
			}
			if poolMap["listener_id"] != "listener-123" {
				t.Errorf("expected listener_id 'listener-123', got %v", poolMap["listener_id"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"pool": map[string]any{
				"id":           "pool-new",
				"name":         "new-pool",
				"protocol":     "HTTP",
				"lb_algorithm": "ROUND_ROBIN",
				"listener_id":  "listener-123",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	pool, err := api.CreatePool("new-pool", "HTTP", "ROUND_ROBIN", "listener-123")
	if err != nil {
		t.Fatalf("CreatePool() error: %v", err)
	}
	if pool.ID != "pool-new" {
		t.Errorf("expected ID 'pool-new', got %q", pool.ID)
	}
	if pool.Name != "new-pool" {
		t.Errorf("expected name 'new-pool', got %q", pool.Name)
	}
}

func TestDeletePool(t *testing.T) {
	const poolID = "pool-del-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/pools/"+poolID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	if err := api.DeletePool(poolID); err != nil {
		t.Fatalf("DeletePool() error: %v", err)
	}
}

// --- Member ---

func TestListMembers(t *testing.T) {
	const poolID = "pool-for-members"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		expectedSuffix := "/v2.0/lbaas/pools/" + poolID + "/members"
		if !strings.HasSuffix(r.URL.Path, expectedSuffix) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"members": []map[string]any{
				{"id": "member-1", "name": "web-1", "address": "10.0.0.1", "protocol_port": 8080, "weight": 1},
				{"id": "member-2", "name": "web-2", "address": "10.0.0.2", "protocol_port": 8080, "weight": 2},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	members, err := api.ListMembers(poolID)
	if err != nil {
		t.Fatalf("ListMembers() error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].ID != "member-1" {
		t.Errorf("expected ID 'member-1', got %q", members[0].ID)
	}
	if members[0].Address != "10.0.0.1" {
		t.Errorf("expected address '10.0.0.1', got %q", members[0].Address)
	}
}

func TestGetMember(t *testing.T) {
	const poolID = "pool-get"
	const memberID = "member-get"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		expectedSuffix := "/v2.0/lbaas/pools/" + poolID + "/members/" + memberID
		if !strings.HasSuffix(r.URL.Path, expectedSuffix) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"member": map[string]any{
				"id":            memberID,
				"name":          "backend-1",
				"address":       "192.168.0.10",
				"protocol_port": 9000,
				"weight":        5,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	member, err := api.GetMember(poolID, memberID)
	if err != nil {
		t.Fatalf("GetMember() error: %v", err)
	}
	if member.ID != memberID {
		t.Errorf("expected ID %q, got %q", memberID, member.ID)
	}
	if member.Weight != 5 {
		t.Errorf("expected weight 5, got %d", member.Weight)
	}
}

func TestCreateMember_WithoutWeight(t *testing.T) {
	const poolID = "pool-create-member"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedSuffix := "/v2.0/lbaas/pools/" + poolID + "/members"
		if !strings.HasSuffix(r.URL.Path, expectedSuffix) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		memberMap, ok := body["member"].(map[string]any)
		if !ok {
			t.Errorf("expected 'member' key in body")
		} else {
			if memberMap["name"] != "node-1" {
				t.Errorf("expected name 'node-1', got %v", memberMap["name"])
			}
			if memberMap["address"] != "10.1.0.5" {
				t.Errorf("expected address '10.1.0.5', got %v", memberMap["address"])
			}
			if memberMap["protocol_port"] != float64(8080) {
				t.Errorf("expected protocol_port 8080, got %v", memberMap["protocol_port"])
			}
			if _, hasWeight := memberMap["weight"]; hasWeight {
				t.Errorf("expected no 'weight' key when weight is nil, got %v", memberMap["weight"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"member": map[string]any{
				"id":            "member-new",
				"name":          "node-1",
				"address":       "10.1.0.5",
				"protocol_port": 8080,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	member, err := api.CreateMember(poolID, "node-1", "10.1.0.5", 8080, nil)
	if err != nil {
		t.Fatalf("CreateMember() error: %v", err)
	}
	if member.ID != "member-new" {
		t.Errorf("expected ID 'member-new', got %q", member.ID)
	}
}

func TestCreateMember_WithWeight(t *testing.T) {
	const poolID = "pool-weight-member"
	weight := 10
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		memberMap, ok := body["member"].(map[string]any)
		if !ok {
			t.Errorf("expected 'member' key in body")
		} else if memberMap["weight"] != float64(10) {
			t.Errorf("expected weight 10, got %v", memberMap["weight"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"member": map[string]any{
				"id":      "member-weighted",
				"name":    "node-2",
				"address": "10.1.0.6",
				"weight":  10,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	member, err := api.CreateMember(poolID, "node-2", "10.1.0.6", 8080, &weight)
	if err != nil {
		t.Fatalf("CreateMember() with weight error: %v", err)
	}
	if member.Weight != 10 {
		t.Errorf("expected weight 10, got %d", member.Weight)
	}
}

func TestDeleteMember(t *testing.T) {
	const poolID = "pool-del-member"
	const memberID = "member-del-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		expectedSuffix := "/v2.0/lbaas/pools/" + poolID + "/members/" + memberID
		if !strings.HasSuffix(r.URL.Path, expectedSuffix) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	if err := api.DeleteMember(poolID, memberID); err != nil {
		t.Fatalf("DeleteMember() error: %v", err)
	}
}

// --- HealthMonitor ---

func TestListHealthMonitors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/healthmonitors") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"healthmonitors": []map[string]any{
				{
					"id":          "hm-1",
					"name":        "tcp-check",
					"type":        "TCP",
					"delay":       5,
					"timeout":     3,
					"max_retries": 3,
				},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	hms, err := api.ListHealthMonitors()
	if err != nil {
		t.Fatalf("ListHealthMonitors() error: %v", err)
	}
	if len(hms) != 1 {
		t.Fatalf("expected 1 health monitor, got %d", len(hms))
	}
	if hms[0].ID != "hm-1" {
		t.Errorf("expected ID 'hm-1', got %q", hms[0].ID)
	}
	if hms[0].Type != "TCP" {
		t.Errorf("expected type 'TCP', got %q", hms[0].Type)
	}
}

func TestGetHealthMonitor(t *testing.T) {
	const hmID = "hm-xyz"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/healthmonitors/"+hmID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"healthmonitor": map[string]any{
				"id":             hmID,
				"name":           "http-check",
				"type":           "HTTP",
				"delay":          10,
				"timeout":        5,
				"max_retries":    2,
				"url_path":       "/health",
				"expected_codes": "200",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	hm, err := api.GetHealthMonitor(hmID)
	if err != nil {
		t.Fatalf("GetHealthMonitor() error: %v", err)
	}
	if hm.ID != hmID {
		t.Errorf("expected ID %q, got %q", hmID, hm.ID)
	}
	if hm.URLPath != "/health" {
		t.Errorf("expected url_path '/health', got %q", hm.URLPath)
	}
	if hm.ExpectedCodes != "200" {
		t.Errorf("expected expected_codes '200', got %q", hm.ExpectedCodes)
	}
}

func TestCreateHealthMonitor_WithURLPathAndCodes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/healthmonitors") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		hmMap, ok := body["healthmonitor"].(map[string]any)
		if !ok {
			t.Errorf("expected 'healthmonitor' key in body")
		} else {
			if hmMap["pool_id"] != "pool-hm" {
				t.Errorf("expected pool_id 'pool-hm', got %v", hmMap["pool_id"])
			}
			if hmMap["name"] != "http-monitor" {
				t.Errorf("expected name 'http-monitor', got %v", hmMap["name"])
			}
			if hmMap["type"] != "HTTP" {
				t.Errorf("expected type 'HTTP', got %v", hmMap["type"])
			}
			if hmMap["delay"] != float64(5) {
				t.Errorf("expected delay 5, got %v", hmMap["delay"])
			}
			if hmMap["timeout"] != float64(3) {
				t.Errorf("expected timeout 3, got %v", hmMap["timeout"])
			}
			if hmMap["max_retries"] != float64(3) {
				t.Errorf("expected max_retries 3, got %v", hmMap["max_retries"])
			}
			if hmMap["url_path"] != "/ping" {
				t.Errorf("expected url_path '/ping', got %v", hmMap["url_path"])
			}
			if hmMap["expected_codes"] != "200,201" {
				t.Errorf("expected expected_codes '200,201', got %v", hmMap["expected_codes"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"healthmonitor": map[string]any{
				"id":             "hm-new",
				"name":           "http-monitor",
				"type":           "HTTP",
				"delay":          5,
				"timeout":        3,
				"max_retries":    3,
				"url_path":       "/ping",
				"expected_codes": "200,201",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	hm, err := api.CreateHealthMonitor("pool-hm", "http-monitor", "HTTP", 5, 3, 3, "/ping", "200,201")
	if err != nil {
		t.Fatalf("CreateHealthMonitor() error: %v", err)
	}
	if hm.ID != "hm-new" {
		t.Errorf("expected ID 'hm-new', got %q", hm.ID)
	}
	if hm.URLPath != "/ping" {
		t.Errorf("expected url_path '/ping', got %q", hm.URLPath)
	}
}

func TestCreateHealthMonitor_WithoutURLPathAndCodes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		hmMap, ok := body["healthmonitor"].(map[string]any)
		if !ok {
			t.Errorf("expected 'healthmonitor' key in body")
		} else {
			if _, hasURLPath := hmMap["url_path"]; hasURLPath {
				t.Errorf("expected no 'url_path' key when urlPath is empty")
			}
			if _, hasCodes := hmMap["expected_codes"]; hasCodes {
				t.Errorf("expected no 'expected_codes' key when expectedCodes is empty")
			}
			if hmMap["type"] != "TCP" {
				t.Errorf("expected type 'TCP', got %v", hmMap["type"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"healthmonitor": map[string]any{
				"id":          "hm-tcp",
				"name":        "tcp-monitor",
				"type":        "TCP",
				"delay":       10,
				"timeout":     5,
				"max_retries": 2,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	hm, err := api.CreateHealthMonitor("pool-tcp", "tcp-monitor", "TCP", 10, 5, 2, "", "")
	if err != nil {
		t.Fatalf("CreateHealthMonitor() without url/codes error: %v", err)
	}
	if hm.ID != "hm-tcp" {
		t.Errorf("expected ID 'hm-tcp', got %q", hm.ID)
	}
	if hm.Type != "TCP" {
		t.Errorf("expected type 'TCP', got %q", hm.Type)
	}
}

func TestDeleteHealthMonitor(t *testing.T) {
	const hmID = "hm-del-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2.0/lbaas/healthmonitors/"+hmID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewLoadBalancerAPI(newTestClient(ts))
	if err := api.DeleteHealthMonitor(hmID); err != nil {
		t.Fatalf("DeleteHealthMonitor() error: %v", err)
	}
}
