package fritzbox

import "strconv"

// Capability enumerates the device capabilities.
type Capability int

// Known (specified) device capabilities.
//
// noinspection GoUnusedConst
const (
	HANFUNCompatibility Capability = iota
	_
	_
	_
	AlertTrigger
	_
	HeatControl
	PowerSensor
	TemperatureSensor
	StateSwitch
	DECTRepeater
	Microphone
	_
	HANFUNUnit
)

type DeviceList struct {
	Devices []Device `xml:"device"`
}

type Device struct {
	Identifier         string `xml:"identifier,attr"`      // A unique ID like AIN, MAC address, etc.
	InternalID         string `xml:"id,attr"`              // Internal device ID of the FRITZ!Box.
	CapabilitiesBitmap string `xml:"functionbitmask,attr"` // Bitmask determining the functionality of the device: bit 6: Comet DECT, HKR, "thermostat", bit 7: energy measurement device, bit 8: temperature sensor, bit 9: switch, bit 10: AVM DECT repeater
	FirmwareVersion    string `xml:"fwversion,attr"`       // Firmware version of the device.
	Manufacturer       string `xml:"manufacturer,attr"`    // Manufacturer of the device, usually set to "AVM".
	ProductName        string `xml:"productname,attr"`     // Name of the product, empty for unknown or undefined devices.
	Present            int    `xml:"present"`              // Device connected (1) or not (0).
	Name               string `xml:"name"`                 // The name of the device. Can be assigned in the web gui of the FRITZ!Box.

	Switch      SwitchInfo      `xml:"switch"`
	Power       PowerInfo       `xml:"powermeter"`
	Temperature TemperatureInfo `xml:"temperature"`

	Thermostat struct {
		Measured   string `xml:"tist"`    // Measured temperature.
		Goal       string `xml:"tsoll"`   // Desired temperature, user controlled.
		Saving     string `xml:"absenk"`  // Energy saving temperature.
		Comfort    string `xml:"komfort"` // Comfortable temperature.
		NextChange struct {
			TimeStamp string `xml:"endperiod"` // Timestamp (epoch time) when the next temperature switch is scheduled.
			Goal      string `xml:"tchange"`   // The temperature to switch to. Same unit convention as in Thermostat.Measured.
		} `xml:"nextchange"` // The next scheduled temperature change.
		Lock       string `xml:"lock"`             // Switch locked (box defined)? 1/0 (empty if not known or if there was an error).
		DeviceLock string `xml:"devicelock"`       // Switch locked (device defined)? 1/0 (empty if not known or if there was an error).
		ErrorCode  string `xml:"errorcode"`        // Error codes: 0 = OK, 1 = ... see https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf.
		BatteryLow string `xml:"batterylow"`       // "0" if the battery is OK, "1" if it is running low on capacity.
		WindowOpen string `xml_:"windowopenactiv"` // "1" if detected an open window (usually turns off heating), "0" if not.
	} `xml:"hkr"`

	AlertSensor struct {
		State string `xml:"state"` // Last transmitted alert state, "0" - no alert, "1" - alert, "" if unknown or upon errors.
	} `xml:"alert"`

	Button struct {
		LastPressedTimestamp string `xml:"lastpressedtimestamp"` // Timestamp (in epoch seconds) when the button was last pressed. "0" or "" if unknown.
	} `xml:"button"`
}

type SwitchInfo struct {
	State      string `xml:"state"`      // Switch state 1/0 on/off (empty if not known or if there was an error).
	Mode       string `xml:"mode"`       // Switch mode manual/automatic (empty if not known or if there was an error).
	Lock       string `xml:"lock"`       // Switch locked (box defined)? 1/0 (empty if not known or if there was an error).
	DeviceLock string `xml:"devicelock"` // Switch locked (device defined)? 1/0 (empty if not known or if there was an error).
}

type PowerInfo struct {
	Power   string `xml:"power"`   // Electric power in milli Watt, refreshed approx every 2 minutes
	Energy  string `xml:"energy"`  // Accumulated power consumption since initial setup
	Voltage string `xml:"voltage"` // Electric voltage in milli Volt, refreshed approx every 2 minutes
}

type TemperatureInfo struct {
	Celsius string `xml:"celsius"` // Temperature measured at the device sensor in units of 0.1 °C. Negative and positive values are possible.
	Offset  string `xml:"offset"`  // Temperature offset (set by the user) in units of 0.1 °C. Negative and positive values are possible.
}

func (i SwitchInfo) IsPoweredOn() bool {
	return i.State == "1"
}

func (i PowerInfo) GetVoltage() float64 {
	f, _ := strconv.ParseFloat(i.Voltage, 64)
	return f / 1000
}

func (i PowerInfo) GetPower() float64 {
	f, _ := strconv.ParseFloat(i.Power, 64)
	return f / 1000
}

func (i PowerInfo) GetEnergy() float64 {
	f, _ := strconv.ParseFloat(i.Energy, 64)
	return f
}

func (i TemperatureInfo) GetCelsius() float64 {
	f, _ := strconv.ParseFloat(i.Celsius, 64)
	return f / 10
}

func (d *Device) CanMeasurePower() bool {
	return d.Has(PowerSensor)
}

func (d *Device) CanMeasureTemperature() bool {
	return d.Has(TemperatureSensor)
}

func (d *Device) IsSwitch() bool {
	return d.Has(StateSwitch)
}

// Has checks the passed capabilities and returns true iff the device supports
// all capabilities.
func (d *Device) Has(cs ...Capability) bool {
	for _, c := range cs {
		b := bitMasked{Functionbitmask: d.CapabilitiesBitmap}.hasMask(1 << uint(c))
		if !b {
			return false
		}
	}
	return true
}

type bitMasked struct {
	Functionbitmask string
}

func (b bitMasked) hasMask(mask int64) bool {
	bitMask, err := strconv.ParseInt(b.Functionbitmask, 10, 64)
	if err != nil {
		return false
	}
	return (bitMask & mask) != 0
}
