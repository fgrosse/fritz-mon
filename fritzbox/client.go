package fritzbox

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

type Client struct {
	Username string
	Password string
	BaseURL  url.URL // must not be a pointer to avoid modifying this URL during our requests

	http    *http.Client
	logger  *zap.Logger
	session Session
}

func New(baseURL, username, password string, logger *zap.Logger) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &Client{
		Username: username,
		Password: password,
		BaseURL:  *u,

		http:   http.DefaultClient,
		logger: logger,
	}, nil
}

func (c *Client) Devices() ([]Device, error) {
	c.logger.Debug("Requesting list of devices")

	var response DeviceList
	err := c.doXMLCommand(&response, "getdevicelistinfos")
	return response.Devices, err
}

func (c *Client) doCommand(cmd string, args ...string) (*bytes.Buffer, error) {
	args, err := c.prepareCommand(cmd, args)
	if err != nil {
		return nil, err
	}
	return c.get("/webservices/homeautoswitch.lua", args...)
}

func (c *Client) doXMLCommand(target interface{}, cmd string, args ...string) error {
	args, err := c.prepareCommand(cmd, args)
	if err != nil {
		return err
	}
	return c.getXML(target, "/webservices/homeautoswitch.lua", args...)
}

func (c *Client) prepareCommand(cmd string, args []string) ([]string, error) {
	if c.session.SID == "" {
		err := c.login()
		if err != nil {
			return nil, fmt.Errorf("authentication error: %w", err)
		}
	}

	return append(args, "sid", c.session.SID, "switchcmd", cmd), nil
}

func (c *Client) Close() error {
	return c.logout()
}
