package api

import (
	"fmt"
	"io"
	"net/http"

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

func (a *ImageAPI) CreateImage(name, diskFormat, containerFormat string) (*model.Image, error) {
	url := fmt.Sprintf("%s/images", a.baseURL())
	body := model.ImageCreateRequest{
		Name:            name,
		DiskFormat:      diskFormat,
		ContainerFormat: containerFormat,
	}
	var img model.Image
	if _, err := a.Client.Post(url, body, &img); err != nil {
		return nil, err
	}
	return &img, nil
}

func (a *ImageAPI) UploadImageFile(id string, reader io.Reader, size int64) error {
	url := fmt.Sprintf("%s/images/%s/file", a.baseURL(), id)
	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Auth-Token", a.Client.Token)
	req.Header.Set("User-Agent", UserAgent)
	req.ContentLength = size

	// Use a dedicated client with no timeout for large file uploads
	uploadClient := &http.Client{}
	resp, err := uploadClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}
