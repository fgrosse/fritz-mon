package fritzbox

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// 20 values representing the last 100 seconds in 20 buckets of 5 seconds each.
type TrafficMonitoringData struct {
	DownstreamInternet      []float64 `json:"ds_bps_curr"`
	DownStreamMedia         []float64 `json:"ds_mc_bps_curr"`
	DownStreamGuest         []float64 `json:"ds_guest_bps_curr"`
	UpstreamRealtime        []float64 `json:"us_realtime_bps_curr"`
	UpstreamHighPriority    []float64 `json:"us_important_bps_curr"`
	UpstreamDefaultPriority []float64 `json:"us_default_bps_curr"`
	UpstreamLowPriority     []float64 `json:"us_background_bps_curr"`
	UpstreamGuest           []float64 `json:"guest_us_bps"`
}

func (c *Client) NetworkStats() (*TrafficMonitoringData, error) {
	sessionID, err := c.getSession()
	if err != nil {
		return nil, err
	}

	resp, err := c.get("/internet/inetstat_monitor.lua",
		"sid", sessionID,
		"myXhr", "1",
		"xhr", "1",
		"useajax", "1",
		"action", "get_graphic",
	)

	if err != nil {
		return nil, errors.Wrap(err, "inetstat_monitor.lua")
	}

	var result []*TrafficMonitoringData
	err = json.NewDecoder(resp).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode response as JSON")
	}

	if len(result) == 0 {
		return nil, errors.Wrap(err, "FRITZ!Box returned no monitoring data")
	}

	return result[0], nil
}
