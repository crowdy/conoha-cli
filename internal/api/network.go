package api

import (
	"fmt"
	"net/http"

	"github.com/crowdy/conoha-cli/internal/model"
)

type NetworkAPI struct {
	Client *Client
}

func NewNetworkAPI(c *Client) *NetworkAPI {
	return &NetworkAPI{Client: c}
}

func (a *NetworkAPI) baseURL() string {
	return a.Client.BaseURL("networking") + "/v2.0"
}

func (a *NetworkAPI) ListNetworks() ([]model.Network, error) {
	url := fmt.Sprintf("%s/networks", a.baseURL())
	var resp model.NetworksResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Networks, nil
}

func (a *NetworkAPI) CreateNetwork(name string) (*model.Network, error) {
	url := fmt.Sprintf("%s/networks", a.baseURL())
	body := map[string]any{"network": map[string]string{"name": name}}
	var resp struct {
		Network model.Network `json:"network"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Network, nil
}

func (a *NetworkAPI) DeleteNetwork(id string) error {
	url := fmt.Sprintf("%s/networks/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *NetworkAPI) ListSubnets() ([]model.Subnet, error) {
	url := fmt.Sprintf("%s/subnets", a.baseURL())
	var resp model.SubnetsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Subnets, nil
}

func (a *NetworkAPI) CreateSubnet(networkID, cidr, name string, ipVersion int) (*model.Subnet, error) {
	url := fmt.Sprintf("%s/subnets", a.baseURL())
	body := map[string]any{
		"subnet": map[string]any{
			"network_id": networkID,
			"cidr":       cidr,
			"ip_version": ipVersion,
			"name":       name,
		},
	}
	var resp struct {
		Subnet model.Subnet `json:"subnet"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Subnet, nil
}

func (a *NetworkAPI) DeleteSubnet(id string) error {
	url := fmt.Sprintf("%s/subnets/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *NetworkAPI) ListPorts() ([]model.Port, error) {
	url := fmt.Sprintf("%s/ports", a.baseURL())
	var resp model.PortsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Ports, nil
}

func (a *NetworkAPI) CreatePort(networkID, name string) (*model.Port, error) {
	url := fmt.Sprintf("%s/ports", a.baseURL())
	body := map[string]any{
		"port": map[string]any{"network_id": networkID, "name": name},
	}
	var resp struct {
		Port model.Port `json:"port"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Port, nil
}

func (a *NetworkAPI) UpdatePort(id string, body map[string]any) error {
	url := fmt.Sprintf("%s/ports/%s", a.baseURL(), id)
	return a.Client.Put(url, map[string]any{"port": body}, nil)
}

func (a *NetworkAPI) DeletePort(id string) error {
	url := fmt.Sprintf("%s/ports/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *NetworkAPI) ListSecurityGroups() ([]model.SecurityGroup, error) {
	url := fmt.Sprintf("%s/security-groups", a.baseURL())
	var resp model.SecurityGroupsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.SecurityGroups, nil
}

func (a *NetworkAPI) GetSecurityGroup(id string) (*model.SecurityGroup, error) {
	url := fmt.Sprintf("%s/security-groups/%s", a.baseURL(), id)
	var resp struct {
		SecurityGroup model.SecurityGroup `json:"security_group"`
	}
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return &resp.SecurityGroup, nil
}

func (a *NetworkAPI) CreateSecurityGroup(name, description string) (*model.SecurityGroup, error) {
	url := fmt.Sprintf("%s/security-groups", a.baseURL())
	body := map[string]any{
		"security_group": map[string]string{"name": name, "description": description},
	}
	var resp struct {
		SecurityGroup model.SecurityGroup `json:"security_group"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.SecurityGroup, nil
}

func (a *NetworkAPI) UpdateSecurityGroup(id string, body map[string]any) error {
	url := fmt.Sprintf("%s/security-groups/%s", a.baseURL(), id)
	return a.Client.Put(url, map[string]any{"security_group": body}, nil)
}

func (a *NetworkAPI) DeleteSecurityGroup(id string) error {
	url := fmt.Sprintf("%s/security-groups/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *NetworkAPI) ListSecurityGroupRules() ([]model.SecurityGroupRule, error) {
	url := fmt.Sprintf("%s/security-group-rules", a.baseURL())
	var resp model.SecurityGroupRulesResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.SecurityGroupRules, nil
}

func (a *NetworkAPI) CreateSecurityGroupRule(sgID, direction, protocol, ethertype string, portMin, portMax *int, remoteIP string) (*model.SecurityGroupRule, error) {
	url := fmt.Sprintf("%s/security-group-rules", a.baseURL())
	rule := map[string]any{
		"security_group_id": sgID,
		"direction":         direction,
		"ethertype":         ethertype,
	}
	if protocol != "" {
		rule["protocol"] = protocol
	}
	if portMin != nil {
		rule["port_range_min"] = *portMin
	}
	if portMax != nil {
		rule["port_range_max"] = *portMax
	}
	if remoteIP != "" {
		rule["remote_ip_prefix"] = remoteIP
	}
	body := map[string]any{"security_group_rule": rule}
	var resp struct {
		SecurityGroupRule model.SecurityGroupRule `json:"security_group_rule"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.SecurityGroupRule, nil
}

func (a *NetworkAPI) DeleteSecurityGroupRule(id string) error {
	url := fmt.Sprintf("%s/security-group-rules/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *NetworkAPI) ListQoSPolicies() ([]model.QoSPolicy, error) {
	url := fmt.Sprintf("%s/qos/policies", a.baseURL())
	var resp model.QoSPoliciesResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Policies, nil
}

// AttachPort attaches a port to a server (uses compute API).
func (a *NetworkAPI) AttachPort(computeClient *Client, serverID, portID string) error {
	url := fmt.Sprintf("%s/v2.1/servers/%s/os-interface", computeClient.BaseURL("compute"), serverID)
	body := map[string]any{
		"interfaceAttachment": map[string]string{"port_id": portID},
	}
	resp, err := computeClient.Request(http.MethodPost, url, body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// DetachPort detaches a port from a server (uses compute API).
func (a *NetworkAPI) DetachPort(computeClient *Client, serverID, portID string) error {
	url := fmt.Sprintf("%s/v2.1/servers/%s/os-interface/%s", computeClient.BaseURL("compute"), serverID, portID)
	return computeClient.Delete(url)
}
