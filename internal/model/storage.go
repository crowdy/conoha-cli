package model

type Container struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
	Bytes int64  `json:"bytes" yaml:"bytes"`
}

type StorageObject struct {
	Name         string `json:"name" yaml:"name"`
	ContentType  string `json:"content_type" yaml:"content_type"`
	Bytes        int64  `json:"bytes" yaml:"bytes"`
	LastModified string `json:"last_modified" yaml:"last_modified"`
	Hash         string `json:"hash" yaml:"hash"`
}

type AccountInfo struct {
	ContainerCount int   `json:"container_count" yaml:"container_count"`
	ObjectCount    int   `json:"object_count" yaml:"object_count"`
	BytesUsed      int64 `json:"bytes_used" yaml:"bytes_used"`
}
