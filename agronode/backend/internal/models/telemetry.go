package models

import "time"

type TelemetryReading struct {
	DeviceID    string    `json:"deviceId"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Sensors     map[string]float64 `json:"sensors,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}
