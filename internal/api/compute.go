package api

import (
	"fmt"
	"net/http"

	"github.com/crowdy/conoha-cli/internal/model"
)

// ComputeAPI handles compute (server) operations.
type ComputeAPI struct {
	Client *Client
}

func NewComputeAPI(c *Client) *ComputeAPI {
	return &ComputeAPI{Client: c}
}

func (a *ComputeAPI) baseURL() string {
	return a.Client.BaseURL("compute") + "/v2.1"
}

// ListServers returns all servers.
func (a *ComputeAPI) ListServers() ([]model.Server, error) {
	url := fmt.Sprintf("%s/servers/detail", a.baseURL())
	var resp model.ServersResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Servers, nil
}

// GetServer returns a server by ID.
func (a *ComputeAPI) GetServer(id string) (*model.Server, error) {
	url := fmt.Sprintf("%s/servers/%s", a.baseURL(), id)
	var resp model.ServerDetail
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return &resp.Server, nil
}

// FindServer finds a server by ID or name.
// If idOrName looks like a UUID, it tries GetServer first.
// Otherwise, it lists all servers and matches by name.
func (a *ComputeAPI) FindServer(idOrName string) (*model.Server, error) {
	// Try as ID first (UUID-like)
	if len(idOrName) == 36 && idOrName[8] == '-' {
		s, err := a.GetServer(idOrName)
		if err == nil {
			return s, nil
		}
	}

	// Search by name
	servers, err := a.ListServers()
	if err != nil {
		return nil, err
	}
	for i := range servers {
		if servers[i].Name == idOrName {
			return &servers[i], nil
		}
	}

	// Fall back to ID lookup (in case it's a short/non-UUID ID)
	s, err := a.GetServer(idOrName)
	if err != nil {
		return nil, fmt.Errorf("server %q not found", idOrName)
	}
	return s, nil
}

// RenameServer updates the server name.
func (a *ComputeAPI) RenameServer(id, newName string) (*model.Server, error) {
	url := fmt.Sprintf("%s/servers/%s", a.baseURL(), id)
	body := map[string]any{
		"server": map[string]string{"name": newName},
	}
	var resp model.ServerDetail
	if err := a.Client.Put(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Server, nil
}

// CreateServer creates a new server.
func (a *ComputeAPI) CreateServer(req *model.ServerCreateRequest) (*model.Server, error) {
	url := fmt.Sprintf("%s/servers", a.baseURL())
	var resp model.ServerDetail
	if _, err := a.Client.Post(url, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Server, nil
}

// DeleteServer deletes a server.
func (a *ComputeAPI) DeleteServer(id string) error {
	url := fmt.Sprintf("%s/servers/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

// ServerAction performs an action on a server (start, stop, reboot, etc.).
func (a *ComputeAPI) ServerAction(id string, action map[string]any) error {
	url := fmt.Sprintf("%s/servers/%s/action", a.baseURL(), id)
	resp, err := a.Client.Request(http.MethodPost, url, action)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// StartServer starts a stopped server.
func (a *ComputeAPI) StartServer(id string) error {
	return a.ServerAction(id, map[string]any{"os-start": nil})
}

// StopServer stops a running server.
func (a *ComputeAPI) StopServer(id string) error {
	return a.ServerAction(id, map[string]any{"os-stop": nil})
}

// RebootServer reboots a server.
func (a *ComputeAPI) RebootServer(id string, hard bool) error {
	rebootType := "SOFT"
	if hard {
		rebootType = "HARD"
	}
	return a.ServerAction(id, map[string]any{
		"reboot": map[string]string{"type": rebootType},
	})
}

// ResizeServer resizes a server to a new flavor.
func (a *ComputeAPI) ResizeServer(id, flavorID string) error {
	return a.ServerAction(id, map[string]any{
		"resize": map[string]string{"flavorRef": flavorID},
	})
}

// RebuildServer rebuilds a server with a new image.
func (a *ComputeAPI) RebuildServer(id, imageID string) error {
	return a.ServerAction(id, map[string]any{
		"rebuild": map[string]string{"imageRef": imageID},
	})
}

// GetConsole gets the VNC console URL via remote-consoles endpoint (Nova 2.6+).
func (a *ComputeAPI) GetConsole(id string) (*model.RemoteConsoleResponse, error) {
	url := fmt.Sprintf("%s/servers/%s/remote-consoles", a.baseURL(), id)
	body := map[string]any{
		"remote_console": map[string]string{
			"protocol": "vnc",
			"type":     "novnc",
		},
	}
	var resp model.RemoteConsoleResponse
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListFlavors returns all flavors.
func (a *ComputeAPI) ListFlavors() ([]model.Flavor, error) {
	url := fmt.Sprintf("%s/flavors/detail", a.baseURL())
	var resp model.FlavorsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Flavors, nil
}

// GetFlavor returns a flavor by ID.
func (a *ComputeAPI) GetFlavor(id string) (*model.Flavor, error) {
	url := fmt.Sprintf("%s/flavors/%s", a.baseURL(), id)
	var resp model.FlavorDetail
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return &resp.Flavor, nil
}

// ListKeypairs returns all keypairs.
func (a *ComputeAPI) ListKeypairs() ([]model.Keypair, error) {
	url := fmt.Sprintf("%s/os-keypairs", a.baseURL())
	var resp model.KeypairsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	keypairs := make([]model.Keypair, len(resp.Keypairs))
	for i, kw := range resp.Keypairs {
		keypairs[i] = kw.Keypair
	}
	return keypairs, nil
}

// CreateKeypair creates a new keypair.
func (a *ComputeAPI) CreateKeypair(req *model.KeypairCreateRequest) (*model.Keypair, error) {
	url := fmt.Sprintf("%s/os-keypairs", a.baseURL())
	var resp model.KeypairWrapper
	if _, err := a.Client.Post(url, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Keypair, nil
}

// DeleteKeypair deletes a keypair.
func (a *ComputeAPI) DeleteKeypair(name string) error {
	url := fmt.Sprintf("%s/os-keypairs/%s", a.baseURL(), name)
	return a.Client.Delete(url)
}

// AttachVolume attaches a volume to a server.
func (a *ComputeAPI) AttachVolume(serverID, volumeID string) error {
	url := fmt.Sprintf("%s/servers/%s/os-volume_attachments", a.baseURL(), serverID)
	body := map[string]any{
		"volumeAttachment": map[string]string{"volumeId": volumeID},
	}
	resp, err := a.Client.Request(http.MethodPost, url, body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// DetachVolume detaches a volume from a server.
func (a *ComputeAPI) DetachVolume(serverID, volumeID string) error {
	url := fmt.Sprintf("%s/servers/%s/os-volume_attachments/%s", a.baseURL(), serverID, volumeID)
	return a.Client.Delete(url)
}

// GetServerMetadata returns server metadata.
func (a *ComputeAPI) GetServerMetadata(id string) (map[string]string, error) {
	url := fmt.Sprintf("%s/servers/%s/metadata", a.baseURL(), id)
	var resp struct {
		Metadata map[string]string `json:"metadata"`
	}
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Metadata, nil
}
