package client

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/log"
)

// DandelionClient client interfaces
type DandelionClient struct {
	URL          string
	c            *websocket.Conn
	wsLock       *sync.Mutex
	lastStatuses map[int]map[string]interface{}
}

// DandelionResponse is the default dandelion restful API response structure
type DandelionResponse struct {
	Code int             `json:"code"`
	Info json.RawMessage `json:"info"`
}

// InstanceStatus is current instance status
type InstanceStatus int

const (
	// APIPrefix is the prefix for the API URL
	APIPrefix = "/api/v1"
)

// status
const (
	StatusOffline InstanceStatus = iota
	StatusChecking
	StatusSyncing
	StatusSuccess
	StatusError
)

// NewDandelionClient create new dandelion client instance
func NewDandelionClient(serverURL string) (*DandelionClient, error) {
	_, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	c := &DandelionClient{
		URL:          serverURL,
		lastStatuses: make(map[int]map[string]interface{}),
	}
	err = c.initWebSocket()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *DandelionClient) initWebSocket() error {
	u, _ := url.Parse(c.URL)

	// websocket connect
	u.Scheme = "ws"
	u.Path = "/connect/push"

	headers := http.Header{}
	headers.Add("User-Agent", UserAgent)

	client, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		return err
	}
	c.wsLock = new(sync.Mutex)
	go func() {
		// TODO: add context
		for {
			time.Sleep(time.Minute * 2)

			var statuses []map[string]interface{}
			for _, v := range c.lastStatuses {
				statuses = append(statuses, v)
			}
			message := app.WSMessage{
				Action:  "ping",
				Payload: statuses,
			}

			c.wsLock.Lock()
			err := client.WriteJSON(message)
			c.wsLock.Unlock()
			if err != nil {
				log.LogError.Errorf("websocket ping failed: %v", err)
				client2, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
				if err == nil {
					// reconnected
					log.LogAccess.Debug("websocket reconnected")
					client = client2
					c.wsLock.Lock()
					c.c = client2
					c.wsLock.Unlock()
				}
			}
		}
	}()

	c.c = client

	return nil
}

// Match found best match config from dandelion server
func (c *DandelionClient) Match(clientConfig *app.ClientConfig) (*app.AppConfig, error) {
	apiURI := APIPrefix + "/match/" + clientConfig.AppID

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
	apiURI := APIPrefix + "/list/" + appID + "/tree/" + commitID

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

// GetZipArchive get zip archived commit files
func (c *DandelionClient) GetZipArchive(appID, commitID string) (*zip.Reader, error) {
	apiURI := APIPrefix + "/archive/" + appID + "/" + commitID + ".zip"

	log.LogAccess.Debugf("GET %s", apiURI)

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s%s", c.URL, apiURI),
		nil)
	if err != nil {
		return nil, err
	}

	InitHTTPRequest(req, false)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// close response
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.LogAccess.Debugf("HTTP %s", resp.Status)

	r := bytes.NewReader(body)
	return zip.NewReader(r, r.Size())
}

// Download remote file to local
func (c *DandelionClient) Download(appID, commitID, remotePath, filePath string) error {
	apiURI := APIPrefix + "/list/" + appID + "/tree/" + commitID + "/" + remotePath

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

// SetStatus set instance status
func (c *DandelionClient) SetStatus(cfg *app.ClientConfig, status InstanceStatus, v ...interface{}) error {
	payload := map[string]interface{}{
		"app_id":      cfg.AppID,
		"host":        cfg.Host,
		"instance_id": cfg.InstanceID,
		"status":      status,
	}
	for _, arg := range v {
		switch e := arg.(type) {
		case map[string]interface{}:
			for k, val := range e {
				payload[k] = val
			}
		}
	}
	message := app.WSMessage{
		Action:  "status",
		Payload: payload,
	}
	// use app_id as key, save last status
	c.lastStatuses[cfg.ID] = payload
	log.LogAccess.Debugf("set status: %v", message)

	c.wsLock.Lock()
	defer c.wsLock.Unlock()

	return c.c.WriteJSON(message)
}

// Close connection to dandelion server
func (c *DandelionClient) Close() error {
	return c.c.Close()
}
