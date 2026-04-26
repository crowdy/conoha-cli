package model

import "encoding/json"

// Domain represents a DNS zone. The real ConoHa API returns the identifier
// as `uuid` on /v1/domains, while the field has historically been `id` in
// internal data flow and on the (mocked) /v1/domains/<id> response.
//
// We expose a single `ID` field and accept both shapes on the wire via a
// custom UnmarshalJSON so callers don't have to know which endpoint
// surfaces which name (#170).
type Domain struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Email       string `json:"email" yaml:"email"`
	TTL         int    `json:"ttl" yaml:"ttl"`
	Serial      int    `json:"serial" yaml:"serial"`
	Status      string `json:"status" yaml:"status"`
	Description string `json:"description" yaml:"description"`
}

// UnmarshalJSON accepts either `id` or `uuid` for the identifier. The real
// /v1/domains list response uses `uuid`; the legacy field name `id` is
// preserved on output so existing JSON consumers / table renderers don't
// break.
func (d *Domain) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID          string `json:"id"`
		UUID        string `json:"uuid"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		TTL         int    `json:"ttl"`
		Serial      int    `json:"serial"`
		Status      string `json:"status"`
		Description string `json:"description"`
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	d.ID = w.ID
	if d.ID == "" {
		// Real API returns `uuid` on the list endpoint. Fall through here
		// when the id slot is empty, but never overwrite a populated id —
		// some mock fixtures (and possibly future API versions) include
		// both fields and we treat `id` as canonical.
		d.ID = w.UUID
	}
	d.Name = w.Name
	d.Email = w.Email
	d.TTL = w.TTL
	d.Serial = w.Serial
	d.Status = w.Status
	d.Description = w.Description
	return nil
}

type DomainsResponse struct {
	Domains []Domain `json:"domains"`
}

// Record represents a DNS record. Same `id` vs `uuid` accommodation as
// Domain (#170): the list endpoint surfaces `uuid` in production.
type Record struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Data     string `json:"data" yaml:"data"`
	TTL      int    `json:"ttl" yaml:"ttl"`
	Priority *int   `json:"priority" yaml:"priority"`
}

// UnmarshalJSON accepts either `id` or `uuid` for the identifier; see Domain.
func (r *Record) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID       string `json:"id"`
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Data     string `json:"data"`
		TTL      int    `json:"ttl"`
		Priority *int   `json:"priority"`
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	r.ID = w.ID
	if r.ID == "" {
		r.ID = w.UUID
	}
	r.Name = w.Name
	r.Type = w.Type
	r.Data = w.Data
	r.TTL = w.TTL
	r.Priority = w.Priority
	return nil
}

type RecordsResponse struct {
	Records []Record `json:"records"`
}
