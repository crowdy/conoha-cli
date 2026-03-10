package model

type Volume struct {
	ID                  string            `json:"id" yaml:"id"`
	Name                string            `json:"name" yaml:"name"`
	Status              string            `json:"status" yaml:"status"`
	Size                int               `json:"size" yaml:"size"`
	VolumeType          string            `json:"volume_type" yaml:"volume_type"`
	Description         string            `json:"description" yaml:"description"`
	CreatedAt           FlexTime          `json:"created_at" yaml:"created_at"`
	Bootable            string            `json:"bootable" yaml:"bootable"`
	VolumeImageMetadata map[string]string `json:"volume_image_metadata,omitempty" yaml:"volume_image_metadata,omitempty"`
}

type VolumesResponse struct {
	Volumes []Volume `json:"volumes"`
}

type VolumeDetail struct {
	Volume Volume `json:"volume"`
}

type VolumeCreateRequest struct {
	Volume struct {
		Size        int    `json:"size"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		VolumeType  string `json:"volume_type,omitempty"`
		SnapshotID  string `json:"snapshot_id,omitempty"`
		ImageRef    string `json:"imageRef,omitempty"`
	} `json:"volume"`
}

type VolumeType struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

type VolumeTypesResponse struct {
	VolumeTypes []VolumeType `json:"volume_types"`
}

type Backup struct {
	ID        string   `json:"id" yaml:"id"`
	Name      string   `json:"name" yaml:"name"`
	Status    string   `json:"status" yaml:"status"`
	VolumeID  string   `json:"volume_id" yaml:"volume_id"`
	Size      int      `json:"size" yaml:"size"`
	CreatedAt FlexTime `json:"created_at" yaml:"created_at"`
}

type BackupsResponse struct {
	Backups []Backup `json:"backups"`
}
