package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/crowdy/conoha-cli/internal/model"
)

type ObjectStorageAPI struct {
	Client *Client
}

func NewObjectStorageAPI(c *Client) *ObjectStorageAPI {
	return &ObjectStorageAPI{Client: c}
}

func (a *ObjectStorageAPI) baseURL() string {
	return a.Client.BaseURL("object-storage") + "/v1/AUTH_" + a.Client.TenantID
}

func (a *ObjectStorageAPI) GetAccountInfo() (*model.AccountInfo, error) {
	req, err := http.NewRequest(http.MethodHead, a.baseURL(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	containerCount, _ := strconv.Atoi(resp.Header.Get("X-Account-Container-Count"))
	objectCount, _ := strconv.Atoi(resp.Header.Get("X-Account-Object-Count"))
	bytesUsed, _ := strconv.ParseInt(resp.Header.Get("X-Account-Bytes-Used"), 10, 64)

	return &model.AccountInfo{
		ContainerCount: containerCount,
		ObjectCount:    objectCount,
		BytesUsed:      bytesUsed,
	}, nil
}

func (a *ObjectStorageAPI) ListContainers() ([]model.Container, error) {
	url := a.baseURL() + "?format=json"
	var containers []model.Container
	if err := a.Client.Get(url, &containers); err != nil {
		return nil, err
	}
	return containers, nil
}

func (a *ObjectStorageAPI) CreateContainer(name string) error {
	url := fmt.Sprintf("%s/%s", a.baseURL(), name)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (a *ObjectStorageAPI) DeleteContainer(name string) error {
	url := fmt.Sprintf("%s/%s", a.baseURL(), name)
	return a.Client.Delete(url)
}

func (a *ObjectStorageAPI) ListObjects(container string) ([]model.StorageObject, error) {
	url := fmt.Sprintf("%s/%s?format=json", a.baseURL(), container)
	var objects []model.StorageObject
	if err := a.Client.Get(url, &objects); err != nil {
		return nil, err
	}
	return objects, nil
}

// ListObjectsWithPrefix returns objects in a container filtered by prefix.
func (a *ObjectStorageAPI) ListObjectsWithPrefix(container, prefix string) ([]model.StorageObject, error) {
	url := fmt.Sprintf("%s/%s?format=json&prefix=%s", a.baseURL(), container, prefix)
	var objects []model.StorageObject
	if err := a.Client.Get(url, &objects); err != nil {
		return nil, err
	}
	return objects, nil
}

func (a *ObjectStorageAPI) UploadObject(container, objectName, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	url := fmt.Sprintf("%s/%s/%s", a.baseURL(), container, objectName)
	req, err := http.NewRequest(http.MethodPut, url, file)
	if err != nil {
		return err
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (a *ObjectStorageAPI) DownloadObject(container, objectName, localPath string) error {
	url := fmt.Sprintf("%s/%s/%s", a.baseURL(), container, objectName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func (a *ObjectStorageAPI) DeleteObject(container, objectName string) error {
	url := fmt.Sprintf("%s/%s/%s", a.baseURL(), container, objectName)
	return a.Client.Delete(url)
}

func (a *ObjectStorageAPI) PublishContainer(name string) error {
	url := fmt.Sprintf("%s/%s", a.baseURL(), name)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Container-Read", ".r:*")
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (a *ObjectStorageAPI) UnpublishContainer(name string) error {
	url := fmt.Sprintf("%s/%s", a.baseURL(), name)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Container-Read", "")
	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
