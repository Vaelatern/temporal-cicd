package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL string
	Token   string
	DryRun  bool
	HTTP    *http.Client
}

type RepoRequest struct {
	URL                  string `json:"url"`
	SSHReadingPrivateKey string `json:"ssh-reading-private-key"`
}

type RepoState struct {
	URL                  string `json:"url"`
	Registered           bool   `json:"registered"`
	SSHReadingPublicKey  string `json:"ssh-reading-public-key"`
}

func (c *Client) do(method, path string, body any) (int, []byte, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rdr = bytes.NewReader(b)
	}
	u := strings.TrimRight(c.BaseURL, "/") + path
	req, err := http.NewRequest(method, u, rdr)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	return res.StatusCode, b, err
}

func (c *Client) GetRepo(name string) (*RepoState, error) {
	code, b, err := c.do(http.MethodGet, "/sync/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	if code == http.StatusNotFound {
		return nil, nil
	}
	if code >= 300 {
		return nil, fmt.Errorf("GET /sync/%s: %d %s", name, code, b)
	}
	var st RepoState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (c *Client) PutRepo(name string, req RepoRequest) error {
	if c.DryRun {
		return nil
	}
	code, b, err := c.do(http.MethodPut, "/sync/"+url.PathEscape(name), req)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("PUT /sync/%s: %d %s", name, code, b)
	}
	return nil
}

func (c *Client) PostRepo(name string, req RepoRequest) error {
	if c.DryRun {
		return nil
	}
	code, b, err := c.do(http.MethodPost, "/sync/"+url.PathEscape(name), req)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("POST /sync/%s: %d %s", name, code, b)
	}
	return nil
}

func (c *Client) DeleteRepo(name string) error {
	if c.DryRun {
		return nil
	}
	code, b, err := c.do(http.MethodDelete, "/sync/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	if code == http.StatusNotFound || code == http.StatusNoContent || code < 300 {
		return nil
	}
	return fmt.Errorf("DELETE /sync/%s: %d %s", name, code, b)
}
