package models

import "time"

// DeviceMetadata stores optional device metadata.
type DeviceMetadata struct {
	Battery        *float64          `json:"battery,omitempty"`
	SignalStrength *float64          `json:"signalStrength,omitempty"`
	Hardware       map[string]string `json:"hardware,omitempty"`
}

type DeviceShadowState map[string]any

// Device represents an IoT device in the system
type Device struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	DeviceID       string `gorm:"column:device_id;uniqueIndex;not null" json:"deviceId" binding:"required"`

	OrganizationID *uint `gorm:"column:organization_id" json:"organizationId,omitempty"`
	ZoneID         *uint `gorm:"column:asset_unit_id" json:"zoneId,omitempty"`

	DeviceType string `gorm:"column:device_type;not null;default:'publisher'" json:"deviceType"`
	Status     string `gorm:"not null;default:'unknown'" json:"status"`
	RegistrationStatus string `gorm:"column:registration_status;not null;default:'active'" json:"registrationStatus"`

	FirmwareVersion string `gorm:"column:firmware_version" json:"firmwareVersion,omitempty"`

	// TLS / provisioning
	ProvisioningStatus string `gorm:"column:provisioning_status;not null;default:'pending'" json:"provisioningStatus"`
	APIKeyHash string `gorm:"column:api_key_hash" json:"-"`
	ProvisioningTokenHash string `gorm:"column:provisioning_token_hash" json:"-"`
	CertificateSerial  string `gorm:"column:certificate_serial;uniqueIndex" json:"-"`
	CertificateExpiresAt *time.Time `gorm:"column:certificate_expires_at" json:"certificateExpiresAt,omitempty"`

	Sensors []Sensor `gorm:"foreignKey:DeviceID;references:DeviceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"sensors,omitempty"`

	Metadata          DeviceMetadata    `gorm:"column:metadata;type:jsonb;serializer:json" json:"metadata,omitempty"`
	Tags              []string          `gorm:"column:tags;type:jsonb;serializer:json" json:"tags,omitempty"`
	DesiredState      DeviceShadowState `gorm:"column:desired_state;type:jsonb;serializer:json" json:"desiredState,omitempty"`
	ReportedState     DeviceShadowState `gorm:"column:reported_state;type:jsonb;serializer:json" json:"reportedState,omitempty"`
	DiscoveredSensors []string           `gorm:"column:discovered_sensors;type:jsonb;serializer:json" json:"discoveredSensors,omitempty"`

	LastSeen  *time.Time `gorm:"column:last_seen" json:"lastSeen,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`
}

// TableName specifies the table name for Device model
func (Device) TableName() string {
	return "devices"
}

// DeviceStatus constants for valid device statuses
const (
	DeviceTypeUnknown   = "unknown"
	DeviceTypePublisher = "publisher"
	DeviceTypeReceiver  = "receiver"

	DeviceStatusUnknown = "unknown"
	DeviceStatusOnline  = "online"
	DeviceStatusOffline = "offline"

	RegistrationStatusActive   = "active"
	RegistrationStatusInactive = "inactive"
	RegistrationStatusRevoked  = "revoked"
)
