package models

import "time"

type TelemetryReading struct {
	DeviceID    string    `json:"deviceId"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	CreatedAt   time.Time `json:"createdAt"`
}
