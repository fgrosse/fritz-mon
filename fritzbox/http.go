package fritzbox

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

func (c *Client) getXML(ctx context.Context, target interface{}, reqPath string, args ...string) error {
	resp, err := c.get(ctx, reqPath, args...)
	if err != nil {
		return err
	}

	err = xml.NewDecoder(resp).Decode(target)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP response: %w", err)
	}

	return nil
}

func (c *Client) get(ctx context.Context, reqPath string, args ...string) (*bytes.Buffer, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("bad number of query arguments (must be a factor of 2)")
	}

	params := url.Values{}
	for i := 0; i < len(args); i += 2 {
		key, val := args[i], args[i+1]
		params.Add(key, val)
	}

	reqURL := c.BaseURL
	reqURL.Path = path.Join(c.BaseURL.Path, reqPath)
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}

	req = req.WithContext(ctx)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad HTTP status code: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	return bytes.NewBuffer(body), nil
}
