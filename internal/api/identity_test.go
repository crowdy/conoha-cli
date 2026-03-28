package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/credentials") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"credentials": []map[string]any{
				{"id": "cred-1", "type": "ec2", "user_id": "user-a", "blob": `{"access":"key1"}`},
				{"id": "cred-2", "type": "ec2", "user_id": "user-b", "blob": `{"access":"key2"}`},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	creds, err := api.ListCredentials()
	if err != nil {
		t.Fatalf("ListCredentials() error: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(creds))
	}
	if creds[0].ID != "cred-1" {
		t.Errorf("expected ID 'cred-1', got %q", creds[0].ID)
	}
	if creds[0].Type != "ec2" {
		t.Errorf("expected type 'ec2', got %q", creds[0].Type)
	}
	if creds[1].UserID != "user-b" {
		t.Errorf("expected user_id 'user-b', got %q", creds[1].UserID)
	}
}

func TestGetCredential(t *testing.T) {
	const credID = "cred-abc-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/credentials/"+credID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"credential": map[string]any{
				"id":      credID,
				"type":    "ec2",
				"user_id": "user-xyz",
				"blob":    `{"access":"mykey"}`,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	cred, err := api.GetCredential(credID)
	if err != nil {
		t.Fatalf("GetCredential() error: %v", err)
	}
	if cred.ID != credID {
		t.Errorf("expected ID %q, got %q", credID, cred.ID)
	}
	if cred.UserID != "user-xyz" {
		t.Errorf("expected user_id 'user-xyz', got %q", cred.UserID)
	}
	if cred.Blob != `{"access":"mykey"}` {
		t.Errorf("unexpected blob: %q", cred.Blob)
	}
}

func TestCreateCredential(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/credentials") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		cred, ok := body["credential"].(map[string]any)
		if !ok {
			t.Errorf("expected 'credential' key in body")
		} else {
			if cred["type"] != "ec2" {
				t.Errorf("expected type 'ec2', got %v", cred["type"])
			}
			if cred["blob"] != `{"access":"newkey"}` {
				t.Errorf("unexpected blob: %v", cred["blob"])
			}
			if cred["user_id"] != "user-new" {
				t.Errorf("expected user_id 'user-new', got %v", cred["user_id"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"credential": map[string]any{
				"id":      "cred-new-id",
				"type":    "ec2",
				"user_id": "user-new",
				"blob":    `{"access":"newkey"}`,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	cred, err := api.CreateCredential("ec2", `{"access":"newkey"}`, "user-new")
	if err != nil {
		t.Fatalf("CreateCredential() error: %v", err)
	}
	if cred.ID != "cred-new-id" {
		t.Errorf("expected ID 'cred-new-id', got %q", cred.ID)
	}
	if cred.Type != "ec2" {
		t.Errorf("expected type 'ec2', got %q", cred.Type)
	}
}

func TestDeleteCredential(t *testing.T) {
	const credID = "cred-del-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/credentials/"+credID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	if err := api.DeleteCredential(credID); err != nil {
		t.Fatalf("DeleteCredential() error: %v", err)
	}
}

func TestListSubUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/users") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"users": []map[string]any{
				{"id": "user-1", "name": "alice", "enabled": true, "domain_id": "domain-x"},
				{"id": "user-2", "name": "bob", "enabled": false, "domain_id": "domain-x"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	users, err := api.ListSubUsers()
	if err != nil {
		t.Fatalf("ListSubUsers() error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].ID != "user-1" {
		t.Errorf("expected ID 'user-1', got %q", users[0].ID)
	}
	if users[0].Name != "alice" {
		t.Errorf("expected name 'alice', got %q", users[0].Name)
	}
	if users[1].Enabled {
		t.Errorf("expected user-2 to be disabled")
	}
}

func TestCreateSubUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/users") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		user, ok := body["user"].(map[string]any)
		if !ok {
			t.Errorf("expected 'user' key in body")
		} else {
			if user["name"] != "carol" {
				t.Errorf("expected name 'carol', got %v", user["name"])
			}
			if user["password"] != "s3cr3t" {
				t.Errorf("expected password 's3cr3t', got %v", user["password"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]any{
				"id":        "user-new-id",
				"name":      "carol",
				"enabled":   true,
				"domain_id": "domain-x",
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	user, err := api.CreateSubUser("carol", "s3cr3t")
	if err != nil {
		t.Fatalf("CreateSubUser() error: %v", err)
	}
	if user.ID != "user-new-id" {
		t.Errorf("expected ID 'user-new-id', got %q", user.ID)
	}
	if user.Name != "carol" {
		t.Errorf("expected name 'carol', got %q", user.Name)
	}
}

func TestUpdateSubUser(t *testing.T) {
	const userID = "user-upd-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/users/"+userID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		user, ok := body["user"].(map[string]any)
		if !ok {
			t.Errorf("expected 'user' wrapper in body")
		} else {
			if user["enabled"] != false {
				t.Errorf("expected enabled=false, got %v", user["enabled"])
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	if err := api.UpdateSubUser(userID, map[string]any{"enabled": false}); err != nil {
		t.Fatalf("UpdateSubUser() error: %v", err)
	}
}

func TestDeleteSubUser(t *testing.T) {
	const userID = "user-del-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/users/"+userID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	if err := api.DeleteSubUser(userID); err != nil {
		t.Fatalf("DeleteSubUser() error: %v", err)
	}
}

func TestListRoles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/roles") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"roles": []map[string]any{
				{"id": "role-1", "name": "admin"},
				{"id": "role-2", "name": "member"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewIdentityAPI(newTestClient(ts))
	roles, err := api.ListRoles()
	if err != nil {
		t.Fatalf("ListRoles() error: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(roles))
	}
	if roles[0].ID != "role-1" {
		t.Errorf("expected ID 'role-1', got %q", roles[0].ID)
	}
	if roles[1].Name != "member" {
		t.Errorf("expected name 'member', got %q", roles[1].Name)
	}
}
