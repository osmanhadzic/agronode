package models

import (
	"time"

	"gorm.io/datatypes"
)

type SensorTriggerEntity struct {
	ID             uint            `gorm:"primaryKey"`
	DeviceID       string          `gorm:"column:device_id;not null;index:idx_sensor_triggers_device_sensor_id,unique"`
	SensorID       string          `gorm:"column:sensor_id;not null;index:idx_sensor_triggers_device_sensor_id,unique"`
	MinValue       *float64        `gorm:"column:min_value"`
	MaxValue       *float64        `gorm:"column:max_value"`
	TargetDeviceID *string         `gorm:"column:target_device_id"`
	FuzzyConfig    datatypes.JSON  `gorm:"column:fuzzy_config;type:jsonb"`
	UpdatedAt      time.Time       `gorm:"column:updated_at;not null;default:now()"`
}

func (SensorTriggerEntity) TableName() string {
	return "sensor_triggers"
}
