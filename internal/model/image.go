package model

import "time"

type Image struct {
	ID              string    `json:"id" yaml:"id"`
	Name            string    `json:"name" yaml:"name"`
	Status          string    `json:"status" yaml:"status"`
	DiskFormat      string    `json:"disk_format" yaml:"disk_format"`
	ContainerFormat string    `json:"container_format" yaml:"container_format"`
	MinDisk         int       `json:"min_disk" yaml:"min_disk"`
	MinRAM          int       `json:"min_ram" yaml:"min_ram"`
	Size            int64     `json:"size" yaml:"size"`
	Checksum        string    `json:"checksum" yaml:"checksum"`
	Visibility      string    `json:"visibility" yaml:"visibility"`
	Owner           string    `json:"owner" yaml:"owner"`
	CreatedAt       time.Time `json:"created_at" yaml:"created_at"`
}

type ImagesResponse struct {
	Images []Image `json:"images"`
}

type ImageQuota struct {
	ImageCount    int `json:"image_count" yaml:"image_count"`
	ImageMaxCount int `json:"image_max_count" yaml:"image_max_count"`
}

type ImageCreateRequest struct {
	Name            string `json:"name"`
	DiskFormat      string `json:"disk_format"`
	ContainerFormat string `json:"container_format,omitempty"`
}
