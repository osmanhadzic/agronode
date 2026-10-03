package models

import "encoding/json"

type SensorTrigger struct {
	Min            *float64       `json:"min,omitempty"`
	Max            *float64       `json:"max,omitempty"`
	TargetDeviceID string         `json:"targetDeviceId,omitempty"`
	FuzzyConfig    json.RawMessage `json:"fuzzyConfig,omitempty"`
}
