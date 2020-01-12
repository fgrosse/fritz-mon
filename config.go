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
	ListenAddr                string        `yaml:"listen_addr"`                 // base URL at which to expose Prometheus metrics
	DeviceMonitoringInterval  time.Duration `yaml:"device_monitoring_interval"`  // how often to scrape device metrics from the FRITZ!Box API
	NetworkMonitoringInterval time.Duration `yaml:"network_monitoring_interval"` // how often to scrape network metrics from the FRITZ!Box API
	FritzBox                  struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		BaseURL  string `yaml:"base_url"`
	} `yaml:"fritzbox"`
}

func LoadConfiguration(path string, logger *zap.Logger) (Config, error) {
	logger.Info("Loading configuration file", zap.String("path", path))

	conf := DefaultConfig()
	f, err := os.Open(path)
	if err != nil {
		return conf, fmt.Errorf("failed to open config file %w", err)
	}

	dec := yaml.NewDecoder(f)
	dec.SetStrict(true)

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

func DefaultConfig() Config {
	var conf Config
	conf.ListenAddr = "0:0:0:0:3000"
	conf.DeviceMonitoringInterval = 5 * time.Minute
	conf.NetworkMonitoringInterval = 100 * time.Second // Fritzbox returns the values of the last 100 seconds in 20 buckets of 5 seconds
	conf.FritzBox.BaseURL = "http://fritz.box"
	return conf
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
	if c.DeviceMonitoringInterval == 0 {
		err = multierr.Append(err, fmt.Errorf("device_monitoring_interval cannot be zero"))
	}
	if c.NetworkMonitoringInterval == 0 {
		err = multierr.Append(err, fmt.Errorf("network_monitoring_interval cannot be zero"))
	}
	if c.FritzBox.BaseURL == "" {
		err = multierr.Append(err, fmt.Errorf("FRITZ!Box base URL cannot be empty"))
	}

	return err
}
