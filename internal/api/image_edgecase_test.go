package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge cases for FindImage that complement TestImageFindImage:
//   - case-insensitive substring matching
//   - exact-match precedence over substring match (both active)
//   - empty-string input is rejected up front (not silently matched)
//   - exact name match against a non-active image is NOT returned
//     (boot path would otherwise fail later with an opaque API error)
func TestImageFindImage_EdgeCases(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "id-1", "name": "Ubuntu-24.04-LTS", "status": "active"},
				{"id": "id-2", "name": "centos-stream-9", "status": "active"},
				// exact-vs-substring precedence
				{"id": "id-3", "name": "ubuntu", "status": "active"},
				{"id": "id-4", "name": "ubuntu-server-edge", "status": "active"},
				// killed image with a clean exact-match name
				{"id": "id-5", "name": "old-debian-11", "status": "killed"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)
	api := NewImageAPI(newTestClient(ts))

	t.Run("case-insensitive substring matches", func(t *testing.T) {
		// UBUNTU != "ubuntu" exactly (exact match is case-sensitive) so
		// falls to substring (case-insensitive) → matches 3 images →
		// ambiguous error. Confirms the case-insensitive substring path.
		_, err := api.FindImage("UBUNTU")
		if err == nil {
			t.Fatal("expected ambiguous error for UBUNTU vs 3 ubuntu-* names")
		}
		if !strings.Contains(err.Error(), "ambiguous") {
			t.Errorf("expected ambiguous error, got: %v", err)
		}
		for _, want := range []string{"Ubuntu-24.04-LTS", "ubuntu", "ubuntu-server-edge"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("expected candidate %q in error, got: %v", want, err)
			}
		}
	})

	t.Run("exact match wins over substring", func(t *testing.T) {
		// "ubuntu" matches id-3 exactly. Substring would also match id-1, id-4.
		img, err := api.FindImage("ubuntu")
		if err != nil {
			t.Fatalf("expected exact-match resolution, got error: %v", err)
		}
		if img.ID != "id-3" {
			t.Errorf("expected id-3 (exact match), got %s name=%q", img.ID, img.Name)
		}
	})

	t.Run("empty input rejected", func(t *testing.T) {
		// Without an explicit guard, "" would substring-match every active
		// image — single-image tenants would silently get that image,
		// multi-image tenants would see an "ambiguous" error. Both are
		// confusing; reject up front.
		_, err := api.FindImage("")
		if err == nil {
			t.Fatal("expected error for empty input")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("expected 'empty' in error, got: %v", err)
		}
	})

	t.Run("exact name match on non-active image not returned", func(t *testing.T) {
		// "old-debian-11" exists but is killed. Returning it would let
		// the boot path attempt a doomed create against a half-baked
		// image. Reject as not-found instead.
		_, err := api.FindImage("old-debian-11")
		if err == nil {
			t.Fatal("expected not-found for exact name on killed image")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}
