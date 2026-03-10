package model

type Domain struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Email       string `json:"email" yaml:"email"`
	TTL         int    `json:"ttl" yaml:"ttl"`
	Serial      int    `json:"serial" yaml:"serial"`
	Status      string `json:"status" yaml:"status"`
	Description string `json:"description" yaml:"description"`
}

type DomainsResponse struct {
	Domains []Domain `json:"domains"`
}

type Record struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Data     string `json:"data" yaml:"data"`
	TTL      int    `json:"ttl" yaml:"ttl"`
	Priority *int   `json:"priority" yaml:"priority"`
}

type RecordsResponse struct {
	Records []Record `json:"records"`
}
