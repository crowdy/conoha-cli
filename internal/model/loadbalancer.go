package model

// IDRef represents an association reference with just an ID (used in loadbalancers, listeners, pools arrays).
type IDRef struct {
	ID string `json:"id" yaml:"id"`
}

type LoadBalancer struct {
	ID                 string `json:"id" yaml:"id"`
	Name               string `json:"name" yaml:"name"`
	Description        string `json:"description" yaml:"description"`
	ProvisioningStatus string `json:"provisioning_status" yaml:"provisioning_status"`
	OperatingStatus    string `json:"operating_status" yaml:"operating_status"`
	VipAddress         string `json:"vip_address" yaml:"vip_address"`
}

type LoadBalancersResponse struct {
	LoadBalancers []LoadBalancer `json:"loadbalancers"`
}

type Listener struct {
	ID                 string  `json:"id" yaml:"id"`
	Name               string  `json:"name" yaml:"name"`
	Description        string  `json:"description" yaml:"description"`
	ProvisioningStatus string  `json:"provisioning_status" yaml:"provisioning_status"`
	OperatingStatus    string  `json:"operating_status" yaml:"operating_status"`
	AdminStateUp       bool    `json:"admin_state_up" yaml:"admin_state_up"`
	Protocol           string  `json:"protocol" yaml:"protocol"`
	ProtocolPort       int     `json:"protocol_port" yaml:"protocol_port"`
	ConnectionLimit    int     `json:"connection_limit" yaml:"connection_limit"`
	DefaultPoolID      string  `json:"default_pool_id" yaml:"default_pool_id"`
	Loadbalancers      []IDRef `json:"loadbalancers" yaml:"loadbalancers"`
	ProjectID          string  `json:"project_id" yaml:"project_id"`
	TenantID           string  `json:"tenant_id" yaml:"tenant_id"`
}

type ListenersResponse struct {
	Listeners []Listener `json:"listeners"`
}

type Pool struct {
	ID                 string   `json:"id" yaml:"id"`
	Name               string   `json:"name" yaml:"name"`
	Description        string   `json:"description" yaml:"description"`
	ProvisioningStatus string   `json:"provisioning_status" yaml:"provisioning_status"`
	OperatingStatus    string   `json:"operating_status" yaml:"operating_status"`
	AdminStateUp       bool     `json:"admin_state_up" yaml:"admin_state_up"`
	Protocol           string   `json:"protocol" yaml:"protocol"`
	LBMethod           string   `json:"lb_algorithm" yaml:"lb_algorithm"`
	Loadbalancers      []IDRef  `json:"loadbalancers" yaml:"loadbalancers"`
	Listeners          []IDRef  `json:"listeners" yaml:"listeners"`
	Members            []string `json:"members" yaml:"members"`
	ProjectID          string   `json:"project_id" yaml:"project_id"`
	TenantID           string   `json:"tenant_id" yaml:"tenant_id"`
}

type PoolsResponse struct {
	Pools []Pool `json:"pools"`
}

type Member struct {
	ID                 string `json:"id" yaml:"id"`
	Name               string `json:"name" yaml:"name"`
	Address            string `json:"address" yaml:"address"`
	ProtocolPort       int    `json:"protocol_port" yaml:"protocol_port"`
	Weight             int    `json:"weight" yaml:"weight"`
	OperatingStatus    string `json:"operating_status" yaml:"operating_status"`
	ProvisioningStatus string `json:"provisioning_status" yaml:"provisioning_status"`
	AdminStateUp       bool   `json:"admin_state_up" yaml:"admin_state_up"`
	ProjectID          string `json:"project_id" yaml:"project_id"`
	TenantID           string `json:"tenant_id" yaml:"tenant_id"`
}

type MembersResponse struct {
	Members []Member `json:"members"`
}

type HealthMonitor struct {
	ID                 string  `json:"id" yaml:"id"`
	Name               string  `json:"name" yaml:"name"`
	Type               string  `json:"type" yaml:"type"`
	Delay              int     `json:"delay" yaml:"delay"`
	Timeout            int     `json:"timeout" yaml:"timeout"`
	MaxRetries         int     `json:"max_retries" yaml:"max_retries"`
	URLPath            string  `json:"url_path" yaml:"url_path"`
	ExpectedCodes      string  `json:"expected_codes" yaml:"expected_codes"`
	AdminStateUp       bool    `json:"admin_state_up" yaml:"admin_state_up"`
	Pools              []IDRef `json:"pools" yaml:"pools"`
	ProvisioningStatus string  `json:"provisioning_status" yaml:"provisioning_status"`
	OperatingStatus    string  `json:"operating_status" yaml:"operating_status"`
	ProjectID          string  `json:"project_id" yaml:"project_id"`
	TenantID           string  `json:"tenant_id" yaml:"tenant_id"`
}

type HealthMonitorsResponse struct {
	HealthMonitors []HealthMonitor `json:"healthmonitors"`
}
