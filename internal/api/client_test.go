package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cerrors "github.com/crowdy/conoha-cli/internal/errors"
)

func TestClientGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("expected auth token, got %q", r.Header.Get("X-Auth-Token"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	client := &Client{HTTP: ts.Client(), Token: "test-token"}

	var result map[string]string
	err := client.Get(ts.URL, &result)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected 'ok', got %q", result["status"])
	}
}

func TestClientPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer ts.Close()

	client := &Client{HTTP: ts.Client(), Token: "tok"}

	var result map[string]string
	_, err := client.Post(ts.URL, map[string]string{"name": "test"}, &result)
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if result["id"] != "123" {
		t.Errorf("expected '123', got %q", result["id"])
	}
}

func TestClientDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := &Client{HTTP: ts.Client(), Token: "tok"}
	if err := client.Delete(ts.URL); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestClient404ReturnsNotFoundError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"not found"}}`))
	}))
	defer ts.Close()

	client := &Client{HTTP: ts.Client(), Token: "tok"}
	err := client.Get(ts.URL, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if _, ok := err.(*cerrors.NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestClient401ReturnsAuthError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"unauthorized"}}`))
	}))
	defer ts.Close()

	client := &Client{HTTP: ts.Client(), Token: "bad"}
	err := client.Get(ts.URL, nil)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if _, ok := err.(*cerrors.AuthError); !ok {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestClient500ReturnsAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"code":"500","message":"internal server error"}}`))
	}))
	defer ts.Close()

	client := &Client{HTTP: ts.Client(), Token: "tok"}
	err := client.Get(ts.URL, nil)
	if err == nil {
		t.Fatal("expected error for 500")
	}
	apiErr, ok := err.(*cerrors.APIError)
	if !ok {
		t.Errorf("expected APIError, got %T: %v", err, err)
	} else if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
}

func TestBaseURL(t *testing.T) {
	client := NewClient("c3j1", "tok", "tenant1")
	url := client.BaseURL("compute")
	expected := "https://compute.c3j1.conoha.io"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

func TestBaseURLWithExtServiceMap(t *testing.T) {
	client := NewClient("c3j1", "tok", "tenant1")

	tests := []struct {
		service  string
		expected string
	}{
		{"image", "https://image-service.c3j1.conoha.io"},
		{"load-balancer", "https://lbaas.c3j1.conoha.io"},
		{"compute", "https://compute.c3j1.conoha.io"},
		{"networking", "https://networking.c3j1.conoha.io"},
	}
	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			url := client.BaseURL(tt.service)
			if url != tt.expected {
				t.Errorf("BaseURL(%q) = %q, want %q", tt.service, url, tt.expected)
			}
		})
	}
}

func TestBaseURLWithEndpointOverride(t *testing.T) {
	t.Setenv("CONOHA_ENDPOINT", "https://staging.internal.gmo.jp")
	client := NewClient("c3j1", "tok", "tenant1")
	url := client.BaseURL("compute")
	expected := "https://staging.internal.gmo.jp"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

func TestUserAgentHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != UserAgent {
			t.Errorf("expected User-Agent %q, got %q", UserAgent, ua)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := &Client{HTTP: ts.Client(), Token: "tok"}
	err := client.Get(ts.URL, nil)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
}
