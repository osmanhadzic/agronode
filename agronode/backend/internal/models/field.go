package models

import "time"

type Field struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"column:organization_id;not null;index" json:"organizationId"`
	FarmID         *uint     `gorm:"column:asset_group_id;index" json:"farmId,omitempty"`
	Name           string    `gorm:"column:name;not null" json:"name"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (Field) TableName() string {
	return "asset_sections"
}
