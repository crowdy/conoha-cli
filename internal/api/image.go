package api

import (
	"fmt"

	"github.com/crowdy/conoha-cli/internal/model"
)

type ImageAPI struct {
	Client *Client
}

func NewImageAPI(c *Client) *ImageAPI {
	return &ImageAPI{Client: c}
}

func (a *ImageAPI) baseURL() string {
	return a.Client.BaseURL("image") + "/v2"
}

func (a *ImageAPI) ListImages() ([]model.Image, error) {
	url := fmt.Sprintf("%s/images", a.baseURL())
	var resp model.ImagesResponse
	if err := a.Client.Get(url, &resp); err != nil {
		return nil, err
	}
	return resp.Images, nil
}

func (a *ImageAPI) GetImage(id string) (*model.Image, error) {
	url := fmt.Sprintf("%s/images/%s", a.baseURL(), id)
	var img model.Image
	if err := a.Client.Get(url, &img); err != nil {
		return nil, err
	}
	return &img, nil
}

func (a *ImageAPI) DeleteImage(id string) error {
	url := fmt.Sprintf("%s/images/%s", a.baseURL(), id)
	return a.Client.Delete(url)
}
