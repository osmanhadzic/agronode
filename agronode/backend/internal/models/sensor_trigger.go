package models

import "time"

type SensorTriggerEntity struct {
	ID        uint      `gorm:"primaryKey"`
	DeviceID  string    `gorm:"column:device_id;not null;index:idx_sensor_triggers_device_sensor,unique"`
	Sensor    string    `gorm:"column:sensor;not null;index:idx_sensor_triggers_device_sensor,unique"`
	MinValue  *float64  `gorm:"column:min_value"`
	MaxValue  *float64  `gorm:"column:max_value"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()"`
}

func (SensorTriggerEntity) TableName() string {
	return "sensor_triggers"
}
