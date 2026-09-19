package models

import "time"

type Zone struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"column:organization_id;not null;index" json:"organizationId"`
	FieldID        uint      `gorm:"column:asset_section_id;not null;index" json:"fieldId"`
	Name           string    `gorm:"column:name;not null" json:"name"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (Zone) TableName() string {
	return "asset_units"
}
