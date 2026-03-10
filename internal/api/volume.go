package api

import (
	"fmt"

	"github.com/crowdy/conoha-cli/internal/model"
)

type VolumeAPI struct {
	Client *Client
}

func NewVolumeAPI(c *Client) *VolumeAPI {
	return &VolumeAPI{Client: c}
}

func (a *VolumeAPI) baseURL() string {
	return a.Client.BaseURL("block-storage") + "/v3/" + a.Client.TenantID
}

func (a *VolumeAPI) ListVolumes() ([]model.Volume, error) {
	url := fmt.Sprintf("%s/volumes/detail", a.baseURL())
	var resp model.VolumesResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Volumes, nil
}

func (a *VolumeAPI) GetVolume(id string) (*model.Volume, error) {
	url := fmt.Sprintf("%s/volumes/%s", a.baseURL(), id)
	var resp model.VolumeDetail
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return &resp.Volume, nil
}

func (a *VolumeAPI) CreateVolume(req *model.VolumeCreateRequest) (*model.Volume, error) {
	url := fmt.Sprintf("%s/volumes", a.baseURL())
	var resp model.VolumeDetail
	if _, err := a.Client.Post(url, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Volume, nil
}

func (a *VolumeAPI) UpdateVolume(id string, body map[string]any) error {
	url := fmt.Sprintf("%s/volumes/%s", a.baseURL(), id)
	return a.Client.Put(url, body, nil)
}

func (a *VolumeAPI) DeleteVolume(id string) error {
	url := fmt.Sprintf("%s/volumes/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}

func (a *VolumeAPI) ListVolumeTypes() ([]model.VolumeType, error) {
	url := fmt.Sprintf("%s/types", a.baseURL())
	var resp model.VolumeTypesResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.VolumeTypes, nil
}

func (a *VolumeAPI) ListBackups() ([]model.Backup, error) {
	url := fmt.Sprintf("%s/backups/detail", a.baseURL())
	var resp model.BackupsResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Backups, nil
}

func (a *VolumeAPI) GetBackup(id string) (*model.Backup, error) {
	url := fmt.Sprintf("%s/backups/%s", a.baseURL(), id)
	var resp struct {
		Backup model.Backup `json:"backup"`
	}
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return &resp.Backup, nil
}

func (a *VolumeAPI) RestoreBackup(backupID, volumeID string) error {
	url := fmt.Sprintf("%s/backups/%s/restore", a.baseURL(), backupID)
	body := map[string]any{
		"restore": map[string]string{"volume_id": volumeID},
	}
	return a.Client.Put(url, body, nil)
}
