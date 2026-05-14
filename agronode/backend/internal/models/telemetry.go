package models

import "time"

type TelemetryReading struct {
	DeviceID    string
	Temperature float64
	Humidity    float64
	CreatedAt   time.Time
}
