package models

import "time"

type SensorData struct {
	ID          uint      `gorm:"primaryKey"`
	DeviceID    string    `gorm:"column:device_id;not null;index"`
	Device      Device    `gorm:"foreignKey:DeviceID;references:DeviceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	SensorID    *string   `gorm:"column:sensor_id;not null;index"`
	Sensor      Sensor    `gorm:"foreignKey:DeviceID,SensorID;references:DeviceID,SensorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Temperature float64   `gorm:"not null"`
	Humidity    float64   `gorm:"not null"`
	Sensors     string    `gorm:"type:jsonb;not null;default:'{}' "`
	Meta        string    `gorm:"type:jsonb;not null;default:'{}' "`
	CreatedAt   time.Time `gorm:"column:created_at;not null;default:now();index"`
}

func (SensorData) TableName() string {
	return "sensor_data"
}
