package main

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ListenAddr         string        `yaml:"listen_addr"`         // base URL at which to expose Prometheus metrics
	MonitoringInterval time.Duration `yaml:"monitoring_interval"` // how often to scrape metrics from the FRITZ!Box API
	FritzBox           struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		BaseURL  string `yaml:"base_url"`
	} `yaml:"fritzbox"`
}

func LoadConfiguration(path string, logger *zap.Logger) (Config, error) {
	var conf Config

	logger.Info("Loading configuration file", zap.String("path", path))
	f, err := os.Open(path)
	if err != nil {
		return conf, fmt.Errorf("failed to open config file %w", err)
	}

	dec := yaml.NewDecoder(f)
	dec.SetStrict(true)

	// Setup some default values.
	conf.ListenAddr = "localhost:4000"
	conf.MonitoringInterval = 5 * time.Minute
	conf.FritzBox.BaseURL = "http://fritz.box"

	err = dec.Decode(&conf)
	_ = f.Close()
	if err != nil {
		return conf, fmt.Errorf("failed to parse config file: %w", err)
	}

	err = conf.Validate()
	if err != nil {
		return conf, fmt.Errorf("invalid configuration: %w", err)
	}

	return conf, nil
}

func (c Config) Validate() error {
	var err error

	if c.ListenAddr == "" {
		err = multierr.Append(err, fmt.Errorf("missing listen_addr"))
	}
	if c.FritzBox.Username == "" {
		err = multierr.Append(err, fmt.Errorf("missing fritzbox.username"))
	}
	if c.FritzBox.Username == "" {
		err = multierr.Append(err, fmt.Errorf("missing fritzbox.password"))
	}
	if c.MonitoringInterval == 0 {
		err = multierr.Append(err, fmt.Errorf("monitoring_interval cannot be zero"))
	}
	if c.FritzBox.BaseURL == "" {
		err = multierr.Append(err, fmt.Errorf("FRITZ!Box base URL cannot be empty"))
	}

	return err
}
