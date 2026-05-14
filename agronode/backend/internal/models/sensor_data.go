package models

import "time"

type SensorData struct {
	ID          uint      `gorm:"primaryKey"`
	DeviceID    string    `gorm:"column:device_id;not null;index"`
	Temperature float64   `gorm:"not null"`
	Humidity    float64   `gorm:"not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;default:now();index"`
}

func (SensorData) TableName() string {
	return "sensor_data"
}
