package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"

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

// FindImage resolves an image by UUID, exact name, or unambiguous substring
// match against active image names (#190). Substring matching lets users
// say `--image ubuntu-24.04` and have the CLI pick e.g.
// `vmi-docker-29.2-ubuntu-24.04-amd64` when that's the only candidate.
//
// Resolution order:
//  1. UUID-shaped input → GetImage
//  2. Exact name match → that image
//  3. Single substring match (case-insensitive) → that image
//  4. Multiple substring matches → error listing candidates
//  5. No match → error with hint
func (a *ImageAPI) FindImage(nameOrID string) (*model.Image, error) {
	if looksLikeUUID(nameOrID) {
		return a.GetImage(nameOrID)
	}
	images, err := a.ListImages()
	if err != nil {
		return nil, err
	}
	// Exact name match wins outright (active or otherwise) — even if a
	// fuzzy substring would also hit, an exact equality is unambiguous.
	for i := range images {
		if images[i].Name == nameOrID {
			return &images[i], nil
		}
	}
	// Substring fallback restricted to active images so we don't surface
	// half-baked uploads.
	needle := strings.ToLower(nameOrID)
	var matches []*model.Image
	for i := range images {
		if images[i].Status != "active" {
			continue
		}
		if strings.Contains(strings.ToLower(images[i].Name), needle) {
			matches = append(matches, &images[i])
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("image %q not found (try `conoha image list` to see available images)", nameOrID)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return nil, fmt.Errorf("image %q is ambiguous, matches %d images: %s.\nUse the full name or UUID instead",
			nameOrID, len(matches), strings.Join(names, ", "))
	}
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
