package api

import (
	"fmt"

	"github.com/crowdy/conoha-cli/internal/model"
)

type LoadBalancerAPI struct {
	Client *Client
}

func NewLoadBalancerAPI(c *Client) *LoadBalancerAPI {
	return &LoadBalancerAPI{Client: c}
}

func (a *LoadBalancerAPI) baseURL() string {
	return a.Client.BaseURL("load-balancer") + "/v2.0/lbaas"
}

func (a *LoadBalancerAPI) ListLoadBalancers() ([]model.LoadBalancer, error) {
	url := fmt.Sprintf("%s/loadbalancers", a.baseURL())
	var resp model.LoadBalancersResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.LoadBalancers, nil
}

func (a *LoadBalancerAPI) GetLoadBalancer(id string) (*model.LoadBalancer, error) {
	url := fmt.Sprintf("%s/loadbalancers/%s", a.baseURL(), id)
	var resp struct {
		LoadBalancer model.LoadBalancer `json:"loadbalancer"`
	}
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return &resp.LoadBalancer, nil
}

func (a *LoadBalancerAPI) CreateLoadBalancer(name, subnetID string) (*model.LoadBalancer, error) {
	url := fmt.Sprintf("%s/loadbalancers", a.baseURL())
	body := map[string]any{
		"loadbalancer": map[string]string{
			"name":          name,
			"vip_subnet_id": subnetID,
		},
	}
	var resp struct {
		LoadBalancer model.LoadBalancer `json:"loadbalancer"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.LoadBalancer, nil
}

func (a *LoadBalancerAPI) UpdateLoadBalancer(id string, body map[string]any) error {
	url := fmt.Sprintf("%s/loadbalancers/%s", a.baseURL(), id)
	return a.Client.Put(url, map[string]any{"loadbalancer": body}, nil)
}

func (a *LoadBalancerAPI) DeleteLoadBalancer(id string) error {
	url := fmt.Sprintf("%s/loadbalancers/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *LoadBalancerAPI) ListListeners() ([]model.Listener, error) {
	url := fmt.Sprintf("%s/listeners", a.baseURL())
	var resp model.ListenersResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Listeners, nil
}

func (a *LoadBalancerAPI) CreateListener(name, protocol string, port int, lbID string) (*model.Listener, error) {
	url := fmt.Sprintf("%s/listeners", a.baseURL())
	body := map[string]any{
		"listener": map[string]any{
			"name":            name,
			"protocol":        protocol,
			"protocol_port":   port,
			"loadbalancer_id": lbID,
		},
	}
	var resp struct {
		Listener model.Listener `json:"listener"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Listener, nil
}

func (a *LoadBalancerAPI) DeleteListener(id string) error {
	url := fmt.Sprintf("%s/listeners/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *LoadBalancerAPI) ListPools() ([]model.Pool, error) {
	url := fmt.Sprintf("%s/pools", a.baseURL())
	var resp model.PoolsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Pools, nil
}

func (a *LoadBalancerAPI) CreatePool(name, protocol, lbAlgorithm, listenerID string) (*model.Pool, error) {
	url := fmt.Sprintf("%s/pools", a.baseURL())
	body := map[string]any{
		"pool": map[string]string{
			"name":         name,
			"protocol":     protocol,
			"lb_algorithm": lbAlgorithm,
			"listener_id":  listenerID,
		},
	}
	var resp struct {
		Pool model.Pool `json:"pool"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Pool, nil
}

func (a *LoadBalancerAPI) DeletePool(id string) error {
	url := fmt.Sprintf("%s/pools/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *LoadBalancerAPI) ListMembers(poolID string) ([]model.Member, error) {
	url := fmt.Sprintf("%s/pools/%s/members", a.baseURL(), poolID)
	var resp model.MembersResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func (a *LoadBalancerAPI) CreateMember(poolID, address string, port, weight int) (*model.Member, error) {
	url := fmt.Sprintf("%s/pools/%s/members", a.baseURL(), poolID)
	body := map[string]any{
		"member": map[string]any{
			"address":       address,
			"protocol_port": port,
			"weight":        weight,
		},
	}
	var resp struct {
		Member model.Member `json:"member"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Member, nil
}

func (a *LoadBalancerAPI) DeleteMember(poolID, memberID string) error {
	url := fmt.Sprintf("%s/pools/%s/members/%s", a.baseURL(), poolID, memberID)
	return a.Client.Delete(url)
}

func (a *LoadBalancerAPI) ListHealthMonitors() ([]model.HealthMonitor, error) {
	url := fmt.Sprintf("%s/healthmonitors", a.baseURL())
	var resp model.HealthMonitorsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.HealthMonitors, nil
}

func (a *LoadBalancerAPI) CreateHealthMonitor(poolID, monitorType string, delay, timeout, maxRetries int) (*model.HealthMonitor, error) {
	url := fmt.Sprintf("%s/healthmonitors", a.baseURL())
	body := map[string]any{
		"healthmonitor": map[string]any{
			"pool_id":     poolID,
			"type":        monitorType,
			"delay":       delay,
			"timeout":     timeout,
			"max_retries": maxRetries,
		},
	}
	var resp struct {
		HealthMonitor model.HealthMonitor `json:"healthmonitor"`
	}
	if _, err := a.Client.Post(url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.HealthMonitor, nil
}

func (a *LoadBalancerAPI) DeleteHealthMonitor(id string) error {
	url := fmt.Sprintf("%s/healthmonitors/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}
