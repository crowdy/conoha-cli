package model

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
	ID             string `json:"id" yaml:"id"`
	Name           string `json:"name" yaml:"name"`
	Protocol       string `json:"protocol" yaml:"protocol"`
	ProtocolPort   int    `json:"protocol_port" yaml:"protocol_port"`
	LoadBalancerID string `json:"loadbalancer_id" yaml:"loadbalancer_id"`
}

type ListenersResponse struct {
	Listeners []Listener `json:"listeners"`
}

type Pool struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Protocol string `json:"protocol" yaml:"protocol"`
	LBMethod string `json:"lb_algorithm" yaml:"lb_algorithm"`
}

type PoolsResponse struct {
	Pools []Pool `json:"pools"`
}

type Member struct {
	ID              string `json:"id" yaml:"id"`
	Address         string `json:"address" yaml:"address"`
	ProtocolPort    int    `json:"protocol_port" yaml:"protocol_port"`
	Weight          int    `json:"weight" yaml:"weight"`
	OperatingStatus string `json:"operating_status" yaml:"operating_status"`
}

type MembersResponse struct {
	Members []Member `json:"members"`
}

type HealthMonitor struct {
	ID         string `json:"id" yaml:"id"`
	Type       string `json:"type" yaml:"type"`
	Delay      int    `json:"delay" yaml:"delay"`
	Timeout    int    `json:"timeout" yaml:"timeout"`
	MaxRetries int    `json:"max_retries" yaml:"max_retries"`
	PoolID     string `json:"pool_id" yaml:"pool_id"`
}

type HealthMonitorsResponse struct {
	HealthMonitors []HealthMonitor `json:"healthmonitors"`
}
