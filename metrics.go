package main

import (
	"fmt"
	"sort"

	"github.com/fgrosse/fritz-mon/fritzbox"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Metrics struct {
	IsConnected *prometheus.GaugeVec
	IsPoweredOn *prometheus.GaugeVec
	Temperature *prometheus.GaugeVec
	Power       *prometheus.GaugeVec
	Voltage     *prometheus.GaugeVec
	Energy      *prometheus.GaugeVec

	logger *zap.Logger
}

func NewMetrics(logger *zap.Logger) *Metrics {
	if logger == nil {
		logger = zap.NewNop()
	}

	namespace := "fritzbox"
	subsystem := "home_automation"
	labelNames := []string{"device_name"}
	return &Metrics{
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

func (m *Metrics) Register(r prometheus.Registerer) error {
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

func (m *Metrics) FetchFrom(client *fritzbox.Client) error {
	m.logger.Debug("Fetching metrics from FRITZ!Box API")

	devices, err := client.Devices()
	if err != nil {
		return fmt.Errorf("failed to fetch devices from the FRITZ!Box API: %w", err)
	}

	for _, device := range devices {
		m.collectDeviceMetrics(device)
	}

	return nil
}

func (m *Metrics) collectDeviceMetrics(device fritzbox.Device) {
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
