// Package proxy provides a client for the conoha-proxy Admin API and
// shell-script generators for managing its Docker container lifecycle.
package proxy

import "time"

// Phase mirrors conoha-proxy's externally observable phases.
type Phase string

const (
	PhaseConfigured Phase = "configured"
	PhaseLive       Phase = "live"
	PhaseSwapping   Phase = "swapping"
)

// Target is an upstream instance deployed to a service slot.
type Target struct {
	URL        string    `json:"url"`
	DeployedAt time.Time `json:"deployed_at"`
}

// HealthPolicy mirrors the proxy health_policy object.
type HealthPolicy struct {
	Path               string `json:"path,omitempty"`
	IntervalMs         int    `json:"interval_ms,omitempty"`
	TimeoutMs          int    `json:"timeout_ms,omitempty"`
	HealthyThreshold   int    `json:"healthy_threshold,omitempty"`
	UnhealthyThreshold int    `json:"unhealthy_threshold,omitempty"`
}

// Service is the proxy response body for the /v1/services endpoints.
type Service struct {
	Name           string        `json:"name"`
	Hosts          []string      `json:"hosts"`
	ActiveTarget   *Target       `json:"active_target,omitempty"`
	DrainingTarget *Target       `json:"draining_target,omitempty"`
	DrainDeadline  *time.Time    `json:"drain_deadline,omitempty"`
	HealthPolicy   *HealthPolicy `json:"health_policy,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Phase          Phase         `json:"phase"`
	TLSStatus      string        `json:"tls_status"`
	TLSError       string        `json:"tls_error,omitempty"`
	LastDeployAt   *time.Time    `json:"last_deploy_at,omitempty"`
}

// UpsertRequest is the body of POST /v1/services.
type UpsertRequest struct {
	Name         string        `json:"name"`
	Hosts        []string      `json:"hosts"`
	HealthPolicy *HealthPolicy `json:"health_policy,omitempty"`
}

// DeployRequest is the body of POST /v1/services/{name}/deploy.
type DeployRequest struct {
	TargetURL string `json:"target_url"`
	DrainMs   int    `json:"drain_ms,omitempty"`
}

// RollbackRequest is the body of POST /v1/services/{name}/rollback.
type RollbackRequest struct {
	DrainMs int `json:"drain_ms,omitempty"`
}
