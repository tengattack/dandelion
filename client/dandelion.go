package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/log"
)

// DandelionClient client interfaces
type DandelionClient struct {
	URL string
}

// DandelionResponse is the default dandelion restful API response structure
type DandelionResponse struct {
	Code int             `json:"code"`
	Info json.RawMessage `json:"info"`
}

// NewDandelionClient create new dandelion client instance
func NewDandelionClient(url string) (*DandelionClient, error) {
	return &DandelionClient{URL: url}, nil
}

// Match found best match config from dandelion server
func (c *DandelionClient) Match(clientConfig *app.ClientConfig) (*app.AppConfig, error) {
	apiURI := "/match/" + clientConfig.AppID

	u := url.Values{}
	u.Add("version", clientConfig.Version)
	u.Add("host", clientConfig.Host)
	u.Add("instance_id", clientConfig.InstanceID)

	log.LogAccess.Debugf("GET %s?%s", apiURI, u.Encode())

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s%s?%s", c.URL, apiURI, u.Encode()),
		nil)
	if err != nil {
		return nil, err
	}

	var resp DandelionResponse
	err = DoHTTPRequest(req, true, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(string(resp.Info))
	}

	var info struct {
		AppID  string        `json:"app_id"`
		Config app.AppConfig `json:"config"`
	}

	err = json.Unmarshal(resp.Info, &info)
	if err != nil {
		return nil, err
	}

	return &info.Config, nil
}

// ListFiles list files for specified app id & commit id
func (c *DandelionClient) ListFiles(appID string, commitID string) ([]string, error) {
	apiURI := "/list/" + appID + "/tree/" + commitID

	log.LogAccess.Debugf("GET %s", apiURI)

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s%s", c.URL, apiURI),
		nil)
	if err != nil {
		return nil, err
	}

	var resp DandelionResponse
	err = DoHTTPRequest(req, true, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(string(resp.Info))
	}

	var info struct {
		AppID    string   `json:"app_id"`
		CommitID string   `json:"commit_id"`
		Files    []string `json:"files"`
	}
	err = json.Unmarshal(resp.Info, &info)
	if err != nil {
		return nil, err
	}

	return info.Files, nil
}

// Download remote file to local
func (c *DandelionClient) Download(appID, commitID, remotePath, filePath string) error {
	apiURI := "/list/" + appID + "/tree/" + commitID + "/" + remotePath

	log.LogAccess.Debugf("GET %s", apiURI)

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s%s", c.URL, apiURI),
		nil)
	if err != nil {
		return err
	}

	InitHTTPRequest(req, false)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	// close response
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.LogAccess.Debugf("HTTP %s\n%s", resp.Status, body)

	if resp.StatusCode != 200 {
		var resp DandelionResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return err
		}
		return errors.New(string(resp.Info))
	}

	err = os.MkdirAll(path.Dir(filePath), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	_, err = f.Write(body)
	return err
}
