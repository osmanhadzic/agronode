package models

import "time"

const (
	IrrigationExecutionStatusSucceeded = "succeeded"
	IrrigationExecutionStatusFailed    = "failed"
)

type IrrigationExecution struct {
	ID           uint64    `gorm:"primaryKey"`
	RuleID       string    `gorm:"column:rule_id;not null"`
	AssetUnitID  *uint     `gorm:"column:asset_unit_id"`
	DeviceID     string    `gorm:"column:device_id;not null"`
	TriggeredAt  time.Time `gorm:"column:triggered_at;not null"`
	Reason       string    `gorm:"column:reason;not null"`
	Status       string    `gorm:"column:status;not null"`
	ErrorMessage *string   `gorm:"column:error_message"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;default:now()"`
}

func (IrrigationExecution) TableName() string {
	return "irrigation_executions"
}
