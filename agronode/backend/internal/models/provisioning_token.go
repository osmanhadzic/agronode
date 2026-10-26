package models

import "time"

// ProvisioningToken is a one-time registration grant. Only the SHA-256 hash of the token is stored.
type ProvisioningToken struct {
	ID               uint       `gorm:"primaryKey"`
	TokenHash        string     `gorm:"column:token_hash;not null;uniqueIndex"`
	OrganizationID   uint       `gorm:"column:organization_id;not null"`
	AssetUnitID      *uint      `gorm:"column:asset_unit_id"`
	CreatedBy        string     `gorm:"column:created_by;not null"`
	ExpiresAt        time.Time  `gorm:"column:expires_at;not null"`
	ConsumedAt       *time.Time `gorm:"column:consumed_at"`
	ConsumedDeviceID *string    `gorm:"column:consumed_device_id"`
	RevokedAt        *time.Time `gorm:"column:revoked_at"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null"`
}

func (ProvisioningToken) TableName() string {
	return "provisioning_tokens"
}
