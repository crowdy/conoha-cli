package api

import (
	"fmt"

	"github.com/crowdy/conoha-cli/internal/model"
)

type DNSAPI struct {
	Client *Client
}

func NewDNSAPI(c *Client) *DNSAPI {
	return &DNSAPI{Client: c}
}

func (a *DNSAPI) baseURL() string {
	return a.Client.BaseURL("dns-service") + "/v1"
}

func (a *DNSAPI) ListDomains() ([]model.Domain, error) {
	url := fmt.Sprintf("%s/domains", a.baseURL())
	var resp model.DomainsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Domains, nil
}

func (a *DNSAPI) GetDomain(id string) (*model.Domain, error) {
	url := fmt.Sprintf("%s/domains/%s", a.baseURL(), id)
	var domain model.Domain
	if err := a.Client.Get(url, &domain); err != nil {
		return nil, err
	}
	return &domain, nil
}

func (a *DNSAPI) CreateDomain(name, email string, ttl int) (*model.Domain, error) {
	url := fmt.Sprintf("%s/domains", a.baseURL())
	body := map[string]any{
		"name":  name,
		"email": email,
		"ttl":   ttl,
	}
	var domain model.Domain
	if _, err := a.Client.Post(url, body, &domain); err != nil {
		return nil, err
	}
	return &domain, nil
}

func (a *DNSAPI) UpdateDomain(id string, body map[string]any) error {
	url := fmt.Sprintf("%s/domains/%s", a.baseURL(), id)
	return a.Client.Put(url, body, nil)
}

func (a *DNSAPI) DeleteDomain(id string) error {
	url := fmt.Sprintf("%s/domains/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *DNSAPI) ListRecords(domainID string) ([]model.Record, error) {
	url := fmt.Sprintf("%s/domains/%s/records", a.baseURL(), domainID)
	var resp model.RecordsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Records, nil
}

func (a *DNSAPI) CreateRecord(domainID, name, recordType, data string, ttl int, priority *int) (*model.Record, error) {
	url := fmt.Sprintf("%s/domains/%s/records", a.baseURL(), domainID)
	body := map[string]any{
		"name": name,
		"type": recordType,
		"data": data,
		"ttl":  ttl,
	}
	if priority != nil {
		body["priority"] = *priority
	}
	var record model.Record
	if _, err := a.Client.Post(url, body, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (a *DNSAPI) UpdateRecord(domainID, recordID string, body map[string]any) error {
	url := fmt.Sprintf("%s/domains/%s/records/%s", a.baseURL(), domainID, recordID)
	return a.Client.Put(url, body, nil)
}

func (a *DNSAPI) DeleteRecord(domainID, recordID string) error {
	url := fmt.Sprintf("%s/domains/%s/records/%s", a.baseURL(), domainID, recordID)
	return a.Client.Delete(url)
}
