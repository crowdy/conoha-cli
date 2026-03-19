package model

type FlavorRef struct {
	ID string `json:"id"`
}

type Server struct {
	ID        string               `json:"id" yaml:"id"`
	Name      string               `json:"name" yaml:"name"`
	Status    string               `json:"status" yaml:"status"`
	Flavor    FlavorRef            `json:"flavor" yaml:"flavor"`
	ImageID   string               `json:"image_id" yaml:"image_id"`
	TenantID  string               `json:"tenant_id" yaml:"tenant_id"`
	KeyName   string               `json:"key_name" yaml:"key_name"`
	Created   FlexTime             `json:"created" yaml:"created"`
	Updated   FlexTime             `json:"updated" yaml:"updated"`
	Addresses map[string][]Address `json:"addresses" yaml:"addresses"`
	Metadata  map[string]string    `json:"metadata" yaml:"metadata"`
}

type Address struct {
	Addr    string `json:"addr" yaml:"addr"`
	Version int    `json:"version" yaml:"version"`
	Type    string `json:"OS-EXT-IPS:type" yaml:"type"`
}

type ServerDetail struct {
	Server Server `json:"server"`
}

type ServersResponse struct {
	Servers []Server `json:"servers"`
}

type ServerCreateRequest struct {
	Server struct {
		Name               string               `json:"name"`
		FlavorRef          string               `json:"flavorRef"`
		ImageRef           string               `json:"imageRef,omitempty"`
		KeyName            string               `json:"key_name,omitempty"`
		SecurityGroups     []SecurityGroupRef   `json:"security_groups,omitempty"`
		BlockDeviceMapping []BlockDeviceMapping `json:"block_device_mapping_v2,omitempty"`
		AdminPass          string               `json:"adminPass,omitempty"`
		Metadata           map[string]string    `json:"metadata,omitempty"`
	} `json:"server"`
}

type SecurityGroupRef struct {
	Name string `json:"name"`
}

type BlockDeviceMapping struct {
	UUID                string `json:"uuid,omitempty"`
	SourceType          string `json:"source_type"`
	DestinationType     string `json:"destination_type"`
	VolumeSize          int    `json:"volume_size,omitempty"`
	BootIndex           int    `json:"boot_index"`
	DeleteOnTermination bool   `json:"delete_on_termination"`
}

type ServerAction struct {
	Action string `json:"action,omitempty"`
}

type ConsoleResponse struct {
	Console struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"console"`
}

type RemoteConsoleResponse struct {
	RemoteConsole struct {
		Protocol string `json:"protocol"`
		Type     string `json:"type"`
		URL      string `json:"url"`
	} `json:"remote_console"`
}

type Keypair struct {
	Name        string `json:"name" yaml:"name"`
	PublicKey   string `json:"public_key" yaml:"public_key"`
	Fingerprint string `json:"fingerprint" yaml:"fingerprint"`
	PrivateKey  string `json:"private_key,omitempty" yaml:"private_key,omitempty"`
}

type KeypairWrapper struct {
	Keypair Keypair `json:"keypair"`
}

type KeypairsResponse struct {
	Keypairs []KeypairWrapper `json:"keypairs"`
}

type KeypairCreateRequest struct {
	Keypair struct {
		Name      string `json:"name"`
		PublicKey string `json:"public_key,omitempty"`
	} `json:"keypair"`
}
