package model

type Network struct {
	ID     string   `json:"id" yaml:"id"`
	Name   string   `json:"name" yaml:"name"`
	Status string   `json:"status" yaml:"status"`
	Subnets []string `json:"subnets" yaml:"subnets"`
}

type NetworksResponse struct {
	Networks []Network `json:"networks"`
}

type Subnet struct {
	ID        string `json:"id" yaml:"id"`
	Name      string `json:"name" yaml:"name"`
	NetworkID string `json:"network_id" yaml:"network_id"`
	CIDR      string `json:"cidr" yaml:"cidr"`
	IPVersion int    `json:"ip_version" yaml:"ip_version"`
	GatewayIP string `json:"gateway_ip" yaml:"gateway_ip"`
}

type SubnetsResponse struct {
	Subnets []Subnet `json:"subnets"`
}

type Port struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	NetworkID   string    `json:"network_id" yaml:"network_id"`
	Status      string    `json:"status" yaml:"status"`
	MACAddress  string    `json:"mac_address" yaml:"mac_address"`
	FixedIPs    []FixedIP `json:"fixed_ips" yaml:"fixed_ips"`
}

type FixedIP struct {
	SubnetID  string `json:"subnet_id" yaml:"subnet_id"`
	IPAddress string `json:"ip_address" yaml:"ip_address"`
}

type PortsResponse struct {
	Ports []Port `json:"ports"`
}

type SecurityGroup struct {
	ID          string              `json:"id" yaml:"id"`
	Name        string              `json:"name" yaml:"name"`
	Description string              `json:"description" yaml:"description"`
	Rules       []SecurityGroupRule `json:"security_group_rules" yaml:"rules"`
}

type SecurityGroupRule struct {
	ID             string `json:"id" yaml:"id"`
	Direction      string `json:"direction" yaml:"direction"`
	Protocol       string `json:"protocol" yaml:"protocol"`
	PortRangeMin   *int   `json:"port_range_min" yaml:"port_range_min"`
	PortRangeMax   *int   `json:"port_range_max" yaml:"port_range_max"`
	RemoteIPPrefix string `json:"remote_ip_prefix" yaml:"remote_ip_prefix"`
	EtherType      string `json:"ethertype" yaml:"ethertype"`
}

type SecurityGroupsResponse struct {
	SecurityGroups []SecurityGroup `json:"security_groups"`
}

type SecurityGroupRulesResponse struct {
	SecurityGroupRules []SecurityGroupRule `json:"security_group_rules"`
}

type QoSPolicy struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

type QoSPoliciesResponse struct {
	Policies []QoSPolicy `json:"policies"`
}
