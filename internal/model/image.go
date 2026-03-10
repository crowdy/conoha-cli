package model

import "time"

type Image struct {
	ID         string    `json:"id" yaml:"id"`
	Name       string    `json:"name" yaml:"name"`
	Status     string    `json:"status" yaml:"status"`
	MinDisk    int       `json:"min_disk" yaml:"min_disk"`
	MinRAM     int       `json:"min_ram" yaml:"min_ram"`
	Size       int64     `json:"size" yaml:"size"`
	CreatedAt  time.Time `json:"created_at" yaml:"created_at"`
	Visibility string    `json:"visibility" yaml:"visibility"`
}

type ImagesResponse struct {
	Images []Image `json:"images"`
}

type ImageQuota struct {
	ImageCount    int `json:"image_count" yaml:"image_count"`
	ImageMaxCount int `json:"image_max_count" yaml:"image_max_count"`
}
