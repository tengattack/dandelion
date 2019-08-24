package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/tengattack/dandelion/client"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
)

// Client for registry
type Client struct {
	endpoint   string
	username   string
	password   string
	httpClient *http.Client
}

// ListTagsResponse {catalog}/tags/list response
type ListTagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// NewClient creates a new registry client
func NewClient(conf *config.SectionRegistry) *Client {
	c := new(Client)
	c.endpoint = conf.Endpoint
	c.username = conf.Username
	c.password = conf.Password
	c.httpClient = &http.Client{}

	return c
}

// ListTags list tags for catalog
func (c *Client) ListTags(catalog string) (*ListTagsResponse, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", c.endpoint, catalog)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client.InitHTTPRequest(req, false)
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var v ListTagsResponse
	err = json.Unmarshal(body, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
