package fritzbox

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Client struct {
	Username string
	Password string
	BaseURL  url.URL // must not be a pointer to avoid modifying this URL during our requests

	http   *http.Client
	logger *zap.Logger

	mu      sync.Mutex
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

func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	c.logger.Debug("Requesting list of devices")

	var response DeviceList
	err := c.doXMLCommand(ctx, &response, "getdevicelistinfos")
	return response.Devices, err
}

func (c *Client) doCommand(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error) {
	args, err := c.prepareCommand(ctx, cmd, args)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/webservices/homeautoswitch.lua", args...)
}

func (c *Client) doXMLCommand(ctx context.Context, target interface{}, cmd string, args ...string) error {
	args, err := c.prepareCommand(ctx, cmd, args)
	if err != nil {
		return err
	}
	return c.getXML(ctx, target, "/webservices/homeautoswitch.lua", args...)
}

func (c *Client) prepareCommand(ctx context.Context, cmd string, args []string) ([]string, error) {
	sessionID, err := c.getSession(ctx)
	if err != nil {
		return nil, err
	}

	return append(args, "sid", sessionID, "switchcmd", cmd), nil
}

func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return c.logout(ctx)
}
