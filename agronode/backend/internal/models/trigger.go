package models

type SensorTrigger struct {
	Min            *float64 `json:"min,omitempty"`
	Max            *float64 `json:"max,omitempty"`
	TargetDeviceID string   `json:"targetDeviceId,omitempty"`
}
