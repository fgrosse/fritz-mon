package fritzbox

import (
	"context"
	"encoding/json"
	"fmt"
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

func (c *Client) NetworkStats(ctx context.Context) (*TrafficMonitoringData, error) {
	sessionID, err := c.getSession(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/internet/inetstat_monitor.lua",
		"sid", sessionID,
		"myXhr", "1",
		"xhr", "1",
		"useajax", "1",
		"action", "get_graphic",
	)

	if err != nil {
		return nil, fmt.Errorf("inetstat_monitor.lua: %w", err)
	}

	var result []*TrafficMonitoringData
	err = json.NewDecoder(resp).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response as JSON: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("FRITZ!Box returned no monitoring data: %w", err)
	}

	return result[0], nil
}
