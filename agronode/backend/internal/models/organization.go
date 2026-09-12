package models

import "time"

type Organization struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Slug      string    `gorm:"column:slug;uniqueIndex" json:"slug,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (Organization) TableName() string {
	return "organizations"
}
