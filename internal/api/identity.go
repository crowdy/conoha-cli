package api

import (
	"fmt"

	"github.com/crowdy/conoha-cli/internal/model"
)

type IdentityAPI struct {
	Client *Client
}

func NewIdentityAPI(c *Client) *IdentityAPI {
	return &IdentityAPI{Client: c}
}

func (a *IdentityAPI) baseURL() string {
	return a.Client.BaseURL("identity") + "/v3"
}

func (a *IdentityAPI) ListCredentials() ([]model.Credential, error) {
	url := fmt.Sprintf("%s/credentials", a.baseURL())
	var resp model.CredentialsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Credentials, nil
}

func (a *IdentityAPI) GetCredential(id string) (*model.Credential, error) {
	url := fmt.Sprintf("%s/credentials/%s", a.baseURL(), id)
	var resp struct {
		Credential model.Credential `json:"credential"`
	}
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return &resp.Credential, nil
}

func (a *IdentityAPI) CreateCredential(credType, blob, userID string) (*model.Credential, error) {
	url := fmt.Sprintf("%s/credentials", a.baseURL())
	body := map[string]any{
		"credential": map[string]string{
			"type":    credType,
			"blob":    blob,
			"user_id": userID,
		},
	}
	var resp struct {
		Credential model.Credential `json:"credential"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Credential, nil
}

func (a *IdentityAPI) DeleteCredential(id string) error {
	url := fmt.Sprintf("%s/credentials/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *IdentityAPI) ListSubUsers() ([]model.SubUser, error) {
	url := fmt.Sprintf("%s/users", a.baseURL())
	var resp model.SubUsersResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Users, nil
}

func (a *IdentityAPI) CreateSubUser(name, password string) (*model.SubUser, error) {
	url := fmt.Sprintf("%s/users", a.baseURL())
	body := map[string]any{
		"user": map[string]string{
			"name":     name,
			"password": password,
		},
	}
	var resp struct {
		User model.SubUser `json:"user"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}

func (a *IdentityAPI) UpdateSubUser(id string, body map[string]any) error {
	url := fmt.Sprintf("%s/users/%s", a.baseURL(), id)
	return a.Client.Put(url, map[string]any{"user": body}, nil)
}

func (a *IdentityAPI) DeleteSubUser(id string) error {
	url := fmt.Sprintf("%s/users/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *IdentityAPI) ListRoles() ([]model.Role, error) {
	url := fmt.Sprintf("%s/roles", a.baseURL())
	var resp model.RolesResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Roles, nil
}
