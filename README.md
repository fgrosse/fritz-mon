# fritz-mon

[![Build Status](https://secure.travis-ci.org/fgrosse/fritz-mon.png?branch=master)](http://travis-ci.org/fgrosse/fritz-mon)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](https://github.com/fgrosse/fritz-mon/blob/master/LICENSE)

Export various metrics from the AVM FRITZ!Box API as Prometheus metric.

### Usage

```shell
# Cross compile and upload to your raspberry:
$ GOARCH=arm64 go build && scp fritz-mon raspberry:/usr/local/bin/fritz-mon
fritz-mon                                                    100% 12MB   9.4MB/s   00:01

# Log into your Pi:
$ ssh raspberry

# TODO: configure it

# Run it:
$ fritz-mon -config=/etc/fritz-mon.yml -debug
2020-01-04T18:00:24.289+0100	INFO	Loading configuration file	{"path": "/etc/fritz-mon.yml"}
2020-01-04T18:00:24.289+0100	INFO	Starting FRITZ!Box monitoring server	{"listen_addr": "localhost:4000", "fritzbox": "http://fritz.box", "monitoring_interval": "5m0s"}
2020-01-04T18:00:25.290+0100	DEBUG	Fetching metrics from FRITZ!Box API
2020-01-04T18:00:25.290+0100	DEBUG	Requesting list of devices
2020-01-04T18:00:25.432+0100	DEBUG	Authenticating new session at FRITZ!Box API	{"base_url": "http://fritz.box"}
2020-01-04T18:00:25.772+0100	DEBUG	Collected device metrics	{"device_name": "Lichterkette Balkon", "energy_watt_hours_total": 129, "is_connected": 1, "is_powered": 1, "power_watts": 2.77, "temperature_celsius": 0.5, "voltage_volt": 228.916}
…
```

### Why?

Node exporter apparently doesn't track this metric and this was super easy to do by myself.

### Systemd

Once you get the program working you can set it up in a more permanent way by
using the provided [`fritz-mon.service`](fritz-mon.service) file to install the
cross compiled binary as simple systemd service into your Raspberry Pi like this:

```shell
# Upload systemd unit file:
$ scp systemd/fritz-mon.service raspberry:/etc/systemd/system/fritz-mon.service

# Log into your Pi:
$ ssh raspberry

# Enable service to start on boot:
$ systemctl enable fritz-mon.service

# Start fritz-mon service now:
$ systemctl start fritz-mon

# Check its running:
$ systemctl status fritz-mon
● fritz-mon.service - FRITZ!Box Monitoring Service
   Loaded: loaded (/etc/systemd/system/fritz-mon.service; enabled; vendor preset: enabled)
   Active: active (running) since Sat 2020-01-04 19:47:14 CET; 4s ago
 Main PID: 28951 (fritz-mon)
   Memory: 932.0K
   CGroup: /system.slice/fritz-mon.service
           └─28951 /usr/local/bin/fritz-mon -config=/etc/fritz-mon.yml

Jan 04 19:47:14 systemd[1]: Started FRITZ!Box Monitoring Service.
Jan 04 19:47:14 fritz-mon[28951]: 2020-01-04T19:47:14.978+0100        INFO        Loading configuration file        {"path": "/etc/fritz-mon.yml"}
Jan 04 19:47:14 fritz-mon[28951]: 2020-01-04T19:47:14.988+0100        INFO        Starting FRITZ!Box monitoring server        {"listen_addr": "localhost:3000", "fritzbox": "http://fritz.box", "monitoring_interval": "5m0s"}
Jan 04 19:47:14 fritz-mon[28951]: 2020-01-04T19:47:14.990+0100        INFO        If you want to see more verbose log run with -debug
```

### License

© Friedrich Große 2020, distributed under [BSD-3-Clause License](LICENSE).
