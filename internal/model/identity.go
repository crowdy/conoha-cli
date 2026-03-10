package model

type Credential struct {
	ID      string `json:"id" yaml:"id"`
	Type    string `json:"type" yaml:"type"`
	UserID  string `json:"user_id" yaml:"user_id"`
	Blob    string `json:"blob" yaml:"blob"`
}

type CredentialsResponse struct {
	Credentials []Credential `json:"credentials"`
}

type SubUser struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	DomainID string `json:"domain_id" yaml:"domain_id"`
}

type SubUsersResponse struct {
	Users []SubUser `json:"users"`
}

type Role struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

type RolesResponse struct {
	Roles []Role `json:"roles"`
}

type Permission struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

type PermissionsResponse struct {
	Permissions []Permission `json:"permissions"`
}
