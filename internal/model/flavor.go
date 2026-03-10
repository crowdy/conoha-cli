package model

type Flavor struct {
	ID    string `json:"id" yaml:"id"`
	Name  string `json:"name" yaml:"name"`
	RAM   int    `json:"ram" yaml:"ram"`
	VCPUs int    `json:"vcpus" yaml:"vcpus"`
	Disk  int    `json:"disk" yaml:"disk"`
}

type FlavorsResponse struct {
	Flavors []Flavor `json:"flavors"`
}

type FlavorDetail struct {
	Flavor Flavor `json:"flavor"`
}
