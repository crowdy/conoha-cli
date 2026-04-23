// Command e2e-stub is a tiny stand-in for the ConoHa compute API used by
// the E2E harness. It answers just enough of `/compute/v2.1/servers...`
// to let the CLI resolve a server record that points at our docker
// target container. Everything else (identity, volume, image...) is
// bypassed by setting CONOHA_TOKEN, so this server can stay minimal.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type address struct {
	Addr    string `json:"addr"`
	Version int    `json:"version"`
	Type    string `json:"OS-EXT-IPS:type"`
}

type server struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Flavor    map[string]string      `json:"flavor"`
	ImageID   string                 `json:"image_id"`
	TenantID  string                 `json:"tenant_id"`
	KeyName   string                 `json:"key_name"`
	Created   string                 `json:"created"`
	Updated   string                 `json:"updated"`
	Addresses map[string][]address   `json:"addresses"`
	Metadata  map[string]string      `json:"metadata"`
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8790", "listen address")
	serverName := flag.String("server-name", "e2e-target", "fake server name")
	serverIP := flag.String("server-ip", "127.0.0.1", "IPv4 the CLI should SSH to")
	flag.Parse()

	s := server{
		ID:       "e2e00000-0000-0000-0000-000000000001",
		Name:     *serverName,
		Status:   "ACTIVE",
		Flavor:   map[string]string{"id": "e2e-flavor"},
		TenantID: "e2e-tenant",
		KeyName:  "e2e",
		Created:  "2026-04-23T00:00:00Z",
		Updated:  "2026-04-23T00:00:00Z",
		Addresses: map[string][]address{
			"e2e-net": {{Addr: *serverIP, Version: 4, Type: "fixed"}},
		},
		Metadata: map[string]string{"instance_name_tag": *serverName},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/compute/v2.1/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"servers": []server{s}})
	})
	mux.HandleFunc("/compute/v2.1/servers/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/compute/v2.1/servers/")
		if id == s.ID || id == s.Name {
			writeJSON(w, map[string]any{"server": s})
			return
		}
		http.Error(w, `{"error":{"message":"server not found"}}`, http.StatusNotFound)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("stub: unhandled %s %s", r.Method, r.URL.Path)
		http.Error(w, `{"error":{"message":"not implemented in e2e stub"}}`, http.StatusNotImplemented)
	})

	log.Printf("e2e-stub listening on %s (server=%s ip=%s)", *addr, s.Name, *serverIP)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
