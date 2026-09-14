package models

import "time"

type DeviceMeta struct {
	Firmware string `json:"fw,omitempty"`
	IP       string `json:"ip,omitempty"`
	RSSI     int    `json:"rssi,omitempty"`
	Uptime   uint64 `json:"uptime,omitempty"`
}

type TelemetryReading struct {
	DeviceID    string             `json:"deviceId"`
	SensorID    string             `json:"sensorId,omitempty"`
	Temperature float64            `json:"temperature"`
	Humidity    float64            `json:"humidity"`
	Sensors     map[string]float64 `json:"sensors,omitempty"`
	Meta        *DeviceMeta        `json:"meta,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
}
