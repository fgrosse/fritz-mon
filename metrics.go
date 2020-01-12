package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/fgrosse/fritz-mon/fritzbox"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Metrics struct {
	Devices *DeviceMetrics
	Network *NetworkMetrics
}

type DeviceMetrics struct {
	IsConnected *prometheus.GaugeVec
	IsPoweredOn *prometheus.GaugeVec
	Temperature *prometheus.GaugeVec
	Power       *prometheus.GaugeVec
	Voltage     *prometheus.GaugeVec
	Energy      *prometheus.GaugeVec

	logger *zap.Logger
}

type NetworkMetrics struct {
	DownstreamInternet      prometheus.Gauge // ds_bps_curr
	DownStreamMedia         prometheus.Gauge // ds_mc_bps_curr
	DownStreamGuest         prometheus.Gauge // ds_guest_bps_curr
	UpstreamRealtime        prometheus.Gauge // us_realtime_bps_curr
	UpstreamHighPriority    prometheus.Gauge // us_important_bps_curr
	UpstreamDefaultPriority prometheus.Gauge // us_default_bps_curr
	UpstreamLowPriority     prometheus.Gauge // us_background_bps_curr
	UpstreamGuest           prometheus.Gauge // guest_us_bps

	logger *zap.Logger
}

func NewMetrics(logger *zap.Logger) *Metrics {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Metrics{
		Devices: NewDeviceMetrics(logger),
		Network: NewNetworkMetrics(logger),
	}
}

func NewDeviceMetrics(logger *zap.Logger) *DeviceMetrics {
	namespace := "fritzbox"
	subsystem := "home_automation"
	labelNames := []string{"device_name"}
	return &DeviceMetrics{
		logger: logger,
		IsConnected: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "device_connected_bool",
				Help:      "Either 0 or 1 to indicate if the device is currently connected to the FRITZ!Box.",
			},
			labelNames,
		),
		IsPoweredOn: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "is_powered_bool",
				Help:      "Either 0 or 1 to indicate if the device is powered on or off.",
			},
			labelNames,
		),
		Temperature: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "temperature_celsius",
				Help:      "Temperature measured at the device sensor in degree Celsius.",
			},
			labelNames,
		),
		Power: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "power_watts",
				Help:      "Electric power in Watt, refreshed approx every 2 minutes.",
			},
			labelNames,
		),
		Voltage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "voltage_volts",
				Help:      "Electric voltage in Volt, refreshed approx every 2 minutes.",
			},
			labelNames,
		),
		Energy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "energy_watthours_total",
				Help:      "Accumulated power consumption in Watt hours since initial setup.",
			},
			labelNames,
		),
	}
}

func NewNetworkMetrics(logger *zap.Logger) *NetworkMetrics {
	namespace := "fritzbox"
	subsystem := "network"

	return &NetworkMetrics{
		logger: logger,
		DownstreamInternet: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "downstream_inet_bps",
				Help:      "Internet downstream in bits per second.",
			},
		),
		DownStreamMedia: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "downstream_media_bps",
				Help:      "Media downstream in bits per second.",
			},
		),
		DownStreamGuest: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "downstream_guest_bps",
				Help:      "Guest network downstream in bits per second.",
			},
		),
		UpstreamRealtime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "upstream_realtime_bps",
				Help:      "Realtime priority upstream in bits per second.",
			},
		),
		UpstreamHighPriority: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "upstream_important_bps",
				Help:      "High priority upstream in bits per second.",
			},
		),
		UpstreamDefaultPriority: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "upstream_default_bps",
				Help:      "Default priority upstream in bits per second.",
			},
		),
		UpstreamLowPriority: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "upstream_background_bps",
				Help:      "Low priority upstream in bits per second.",
			},
		),
		UpstreamGuest: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "upstream_guest_bps",
				Help:      "Guest network upstream in bits per second.",
			},
		),
	}
}

func (m *Metrics) Register(r prometheus.Registerer) error {
	if err := m.Devices.Register(r); err != nil {
		return err
	}

	if err := m.Network.Register(r); err != nil {
		return err
	}

	return nil
}

func (m *DeviceMetrics) Register(r prometheus.Registerer) error {
	metrics := []prometheus.Collector{
		m.IsPoweredOn,
		m.IsConnected,
		m.Temperature,
		m.Power,
		m.Voltage,
		m.Energy,
	}

	for _, metric := range metrics {
		if err := r.Register(metric); err != nil {
			return err
		}
	}

	return nil
}

func (m *NetworkMetrics) Register(r prometheus.Registerer) error {
	metrics := []prometheus.Collector{
		m.DownstreamInternet,
		m.DownStreamMedia,
		m.DownStreamGuest,
		m.UpstreamRealtime,
		m.UpstreamHighPriority,
		m.UpstreamDefaultPriority,
		m.UpstreamLowPriority,
		m.UpstreamGuest,
	}

	for _, metric := range metrics {
		if err := r.Register(metric); err != nil {
			return err
		}
	}

	return nil
}

func (m *DeviceMetrics) FetchFrom(ctx context.Context, client *fritzbox.Client) error {
	devices, err := client.Devices(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch devices from the FRITZ!Box API: %w", err)
	}

	for _, device := range devices {
		m.collectDeviceMetrics(device)
	}

	return nil
}

func (m *DeviceMetrics) collectDeviceMetrics(device fritzbox.Device) {
	collectedMetrics := map[string]float64{}
	m.IsConnected.WithLabelValues(device.Name).Set(float64(device.Present))
	collectedMetrics["is_connected"] = float64(device.Present)

	if device.CanMeasureTemperature() {
		temp := device.Temperature.GetCelsius()
		m.Temperature.WithLabelValues(device.Name).Set(temp)
		collectedMetrics["temperature_celsius"] = temp
	}

	if device.CanMeasurePower() {
		volt := device.Power.GetVoltage()
		power := device.Power.GetPower()
		energy := device.Power.GetEnergy()

		m.Voltage.WithLabelValues(device.Name).Set(volt)
		collectedMetrics["voltage_volt"] = volt

		m.Power.WithLabelValues(device.Name).Set(power)
		collectedMetrics["power_watts"] = power

		m.Energy.WithLabelValues(device.Name).Set(energy)
		collectedMetrics["energy_watt_hours_total"] = energy
	}

	if device.IsSwitch() {
		isPowered := prometheusBool(device.Switch.IsPoweredOn())
		m.IsPoweredOn.WithLabelValues(device.Name).Set(isPowered)
		collectedMetrics["is_powered"] = isPowered
	}

	logFields := metricsToLogFields(device.Name, collectedMetrics)
	m.logger.Debug("Collected device metrics", logFields...)
}

func (m *NetworkMetrics) FetchFrom(ctx context.Context, client *fritzbox.Client) error {
	stats, err := client.NetworkStats(ctx)
	if err != nil {
		return err
	}

	m.DownstreamInternet.Set(stats.DownstreamInternet[0] * 8)
	m.DownStreamMedia.Set(stats.DownStreamMedia[0] * 8)
	m.DownStreamGuest.Set(stats.DownStreamGuest[0] * 8)
	m.UpstreamRealtime.Set(stats.UpstreamRealtime[0] * 8)
	m.UpstreamHighPriority.Set(stats.UpstreamHighPriority[0] * 8)
	m.UpstreamDefaultPriority.Set(stats.UpstreamDefaultPriority[0] * 8)
	m.UpstreamLowPriority.Set(stats.UpstreamLowPriority[0] * 8)
	m.UpstreamGuest.Set(stats.UpstreamGuest[0] * 8)

	m.logger.Debug("Collected network metrics")
	return nil
}

func prometheusBool(value bool) float64 {
	if value {
		return 1
	}

	return 0
}

func metricsToLogFields(deviceName string, metrics map[string]float64) []zap.Field {
	names := make([]string, 0, len(metrics))
	for name := range metrics {
		names = append(names, name)
	}

	sort.Strings(names)

	logFields := []zap.Field{zap.String("device_name", deviceName)}
	for _, name := range names {
		value := metrics[name]
		logFields = append(logFields, zap.Float64(name, value))
	}

	return logFields
}
