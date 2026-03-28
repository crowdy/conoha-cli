package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/model"
)

func TestDomainList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/domains") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"domains": []map[string]any{
				{"id": "domain-1", "name": "example.com.", "email": "admin@example.com", "ttl": 3600, "status": "ACTIVE"},
				{"id": "domain-2", "name": "test.com.", "email": "admin@test.com", "ttl": 7200, "status": "ACTIVE"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	domains, err := a.ListDomains()
	if err != nil {
		t.Fatalf("ListDomains() error: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0].ID != "domain-1" {
		t.Errorf("expected domain ID 'domain-1', got %q", domains[0].ID)
	}
	if domains[0].Name != "example.com." {
		t.Errorf("expected domain name 'example.com.', got %q", domains[0].Name)
	}
	if domains[1].TTL != 7200 {
		t.Errorf("expected TTL 7200, got %d", domains[1].TTL)
	}
}

func TestDomainGet(t *testing.T) {
	const domainID = "domain-abc"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/domains/"+domainID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     domainID,
			"name":   "example.com.",
			"email":  "admin@example.com",
			"ttl":    3600,
			"serial": 2024010101,
			"status": "ACTIVE",
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	domain, err := a.GetDomain(domainID)
	if err != nil {
		t.Fatalf("GetDomain() error: %v", err)
	}
	if domain.ID != domainID {
		t.Errorf("expected domain ID %q, got %q", domainID, domain.ID)
	}
	if domain.Name != "example.com." {
		t.Errorf("expected name 'example.com.', got %q", domain.Name)
	}
	if domain.TTL != 3600 {
		t.Errorf("expected TTL 3600, got %d", domain.TTL)
	}
}

func TestDomainCreate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/domains") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["name"] != "newdomain.com." {
			t.Errorf("expected name 'newdomain.com.', got %v", body["name"])
		}
		if body["email"] != "admin@newdomain.com" {
			t.Errorf("expected email 'admin@newdomain.com', got %v", body["email"])
		}
		if body["ttl"] != float64(3600) {
			t.Errorf("expected ttl 3600, got %v", body["ttl"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "new-domain-id",
			"name":  "newdomain.com.",
			"email": "admin@newdomain.com",
			"ttl":   3600,
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	domain, err := a.CreateDomain("newdomain.com.", "admin@newdomain.com", 3600)
	if err != nil {
		t.Fatalf("CreateDomain() error: %v", err)
	}
	if domain.ID != "new-domain-id" {
		t.Errorf("expected domain ID 'new-domain-id', got %q", domain.ID)
	}
	if domain.Name != "newdomain.com." {
		t.Errorf("expected name 'newdomain.com.', got %q", domain.Name)
	}
}

func TestDomainUpdate(t *testing.T) {
	const domainID = "domain-upd"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/domains/"+domainID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["ttl"] != float64(7200) {
			t.Errorf("expected ttl 7200, got %v", body["ttl"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	err := a.UpdateDomain(domainID, map[string]any{"ttl": 7200})
	if err != nil {
		t.Fatalf("UpdateDomain() error: %v", err)
	}
}

func TestDomainDelete(t *testing.T) {
	const domainID = "domain-del"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/domains/"+domainID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	err := a.DeleteDomain(domainID)
	if err != nil {
		t.Fatalf("DeleteDomain() error: %v", err)
	}
}

func TestRecordList(t *testing.T) {
	const domainID = "domain-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		expectedPath := "/v1/domains/" + domainID + "/records"
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{"id": "record-1", "name": "www.example.com.", "type": "A", "data": "192.168.1.1", "ttl": 3600},
				{"id": "record-2", "name": "mail.example.com.", "type": "MX", "data": "mail.example.com.", "ttl": 3600, "priority": 10},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	records, err := a.ListRecords(domainID)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "record-1" {
		t.Errorf("expected record ID 'record-1', got %q", records[0].ID)
	}
	if records[0].Type != "A" {
		t.Errorf("expected type 'A', got %q", records[0].Type)
	}
	if records[1].Priority == nil || *records[1].Priority != 10 {
		t.Errorf("expected priority 10 for MX record")
	}
}

func TestRecordCreateWithoutPriority(t *testing.T) {
	const domainID = "domain-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/v1/domains/" + domainID + "/records"
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["name"] != "www.example.com." {
			t.Errorf("expected name 'www.example.com.', got %v", body["name"])
		}
		if body["type"] != "A" {
			t.Errorf("expected type 'A', got %v", body["type"])
		}
		if body["data"] != "192.168.1.1" {
			t.Errorf("expected data '192.168.1.1', got %v", body["data"])
		}
		if _, hasPriority := body["priority"]; hasPriority {
			t.Errorf("priority should not be present when nil")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "record-new",
			"name": "www.example.com.",
			"type": "A",
			"data": "192.168.1.1",
			"ttl":  3600,
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	record, err := a.CreateRecord(domainID, "www.example.com.", "A", "192.168.1.1", 3600, nil)
	if err != nil {
		t.Fatalf("CreateRecord() error: %v", err)
	}
	if record.ID != "record-new" {
		t.Errorf("expected record ID 'record-new', got %q", record.ID)
	}
	if record.Type != "A" {
		t.Errorf("expected type 'A', got %q", record.Type)
	}
}

func TestRecordCreateWithPriority(t *testing.T) {
	const domainID = "domain-123"
	priority := 10
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["type"] != "MX" {
			t.Errorf("expected type 'MX', got %v", body["type"])
		}
		if body["priority"] != float64(10) {
			t.Errorf("expected priority 10, got %v", body["priority"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		p := 10
		resp := model.Record{
			ID:       "record-mx",
			Name:     "mail.example.com.",
			Type:     "MX",
			Data:     "mail.example.com.",
			TTL:      3600,
			Priority: &p,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	record, err := a.CreateRecord(domainID, "mail.example.com.", "MX", "mail.example.com.", 3600, &priority)
	if err != nil {
		t.Fatalf("CreateRecord() with priority error: %v", err)
	}
	if record.ID != "record-mx" {
		t.Errorf("expected record ID 'record-mx', got %q", record.ID)
	}
	if record.Priority == nil || *record.Priority != 10 {
		t.Errorf("expected priority 10, got %v", record.Priority)
	}
}

func TestRecordUpdate(t *testing.T) {
	const domainID = "domain-123"
	const recordID = "record-456"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		expectedPath := "/v1/domains/" + domainID + "/records/" + recordID
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["data"] != "10.0.0.1" {
			t.Errorf("expected data '10.0.0.1', got %v", body["data"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	err := a.UpdateRecord(domainID, recordID, map[string]any{"data": "10.0.0.1"})
	if err != nil {
		t.Fatalf("UpdateRecord() error: %v", err)
	}
}

func TestRecordDelete(t *testing.T) {
	const domainID = "domain-123"
	const recordID = "record-789"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		expectedPath := "/v1/domains/" + domainID + "/records/" + recordID
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	a := NewDNSAPI(newTestClient(ts))
	err := a.DeleteRecord(domainID, recordID)
	if err != nil {
		t.Fatalf("DeleteRecord() error: %v", err)
	}
}
