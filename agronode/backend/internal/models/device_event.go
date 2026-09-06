package models

import "time"

// DeviceStatusEvent represents a device status change event
type DeviceStatusEvent struct {
	DeviceID   string    `json:"deviceId"`
	OldStatus  string    `json:"oldStatus"`
	NewStatus  string    `json:"newStatus"`
	EventType  string    `json:"eventType"`
	Timestamp  time.Time `json:"timestamp"`
}

// Event type constants
const (
	EventDeviceOnline  = "device.online"
	EventDeviceOffline = "device.offline"
)
