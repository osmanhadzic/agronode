package models

import "time"

// Sensor represents a sensor belonging to a device.
type Sensor struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DeviceID  string    `gorm:"column:device_id;not null;index:idx_sensors_device_sensor,unique" json:"deviceId"`
	SensorID  string    `gorm:"column:sensor_id;not null;index:idx_sensors_device_sensor,unique" json:"sensorId"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (Sensor) TableName() string {
	return "sensors"
}
