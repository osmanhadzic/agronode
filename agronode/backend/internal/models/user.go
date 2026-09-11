package models

import "time"

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"column:organization_id;not null;index" json:"organizationId"`
	Email          string    `gorm:"column:email;not null" json:"email"`
	FullName       string    `gorm:"column:full_name" json:"fullName,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

func (User) TableName() string {
	return "users"
}
