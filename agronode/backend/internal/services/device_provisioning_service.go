package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agronode/backend/internal/auth"
	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/tenancy"
	"github.com/google/uuid"
)

const (
	provisioningTokenPrefix = "agp_"
	provisioningTokenBytes  = 32
	maxProvisioningTokenLen = 128
	DefaultProvisioningTTL  = 24 * time.Hour
	MaxProvisioningTTL      = 7 * 24 * time.Hour
)

var (
	ErrProvisioningForbidden = errors.New("not allowed to manage devices in this organization")
	ErrInvalidLifecycle      = errors.New("lifecycle status must be one of active, disabled, revoked")
	ErrProvisioningTokenBad  = repositories.ErrProvisioningTokenInvalid
)

type IssuedProvisioningToken struct {
	Token     string
	ExpiresAt time.Time
}

type DeviceRegistration struct {
	Device       *models.Device
	MQTTUsername string
	MQTTPassword string
}

type DeviceProvisioningService struct {
	repository repositories.ProvisioningRepository
	logger     *slog.Logger
	now        func() time.Time
}

func NewDeviceProvisioningService(repository repositories.ProvisioningRepository, logger *slog.Logger) *DeviceProvisioningService {
	return &DeviceProvisioningService{repository: repository, logger: logger, now: time.Now}
}

func HashProvisioningToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomSecret(prefix string, size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buffer), nil
}

// IssueToken creates a one-time token. The plaintext is returned once and never stored.
func (service *DeviceProvisioningService) IssueToken(ctx context.Context, organizationID uint, assetUnitID *uint, ttl time.Duration, createdBy string) (*IssuedProvisioningToken, error) {
	if organizationID == 0 {
		return nil, errors.Join(ErrDeviceValidation, errors.New("organizationId is required"))
	}
	if scopedOrganizationID, scoped := tenancy.OrganizationIDFromContext(ctx); scoped && scopedOrganizationID != organizationID {
		return nil, ErrProvisioningForbidden
	}
	if ttl <= 0 {
		ttl = DefaultProvisioningTTL
	}
	if ttl > MaxProvisioningTTL {
		return nil, errors.Join(ErrDeviceValidation, fmt.Errorf("ttl must not exceed %s", MaxProvisioningTTL))
	}
	if assetUnitID != nil {
		belongs, err := service.repository.AssetUnitBelongsToOrganization(ctx, *assetUnitID, organizationID)
		if err != nil {
			return nil, err
		}
		if !belongs {
			return nil, errors.Join(ErrDeviceValidation, errors.New("assetUnitId does not belong to the organization"))
		}
	}

	plain, err := randomSecret(provisioningTokenPrefix, provisioningTokenBytes)
	if err != nil {
		return nil, err
	}

	now := service.now().UTC()
	record := &models.ProvisioningToken{
		TokenHash:      HashProvisioningToken(plain),
		OrganizationID: organizationID,
		AssetUnitID:    assetUnitID,
		CreatedBy:      createdBy,
		ExpiresAt:      now.Add(ttl),
		CreatedAt:      now,
	}
	if err := service.repository.CreateToken(ctx, record); err != nil {
		return nil, err
	}

	service.logger.Info("audit", "event", "device.provisioning_token_issued", "organizationId", organizationID, "createdBy", createdBy, "expiresAt", record.ExpiresAt)

	return &IssuedProvisioningToken{Token: plain, ExpiresAt: record.ExpiresAt}, nil
}

// Register creates a device from a valid token. It never returns credentials for an existing device.
func (service *DeviceProvisioningService) Register(ctx context.Context, hardwareID string, token string) (*DeviceRegistration, error) {
	normalizedHardwareID, err := NormalizeHardwareID(hardwareID)
	if err != nil {
		return nil, err
	}

	token = strings.TrimSpace(token)
	if token == "" || len(token) > maxProvisioningTokenLen {
		return nil, ErrProvisioningTokenBad
	}

	deviceUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate device id: %w", err)
	}
	deviceID := deviceUUID.String()

	password, err := randomSecret("", 32)
	if err != nil {
		return nil, err
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash mqtt password: %w", err)
	}

	device, err := service.repository.RegisterDevice(ctx, HashProvisioningToken(token), service.now().UTC(), repositories.ProvisionedDevice{
		DeviceID:         deviceID,
		HardwareID:       normalizedHardwareID,
		MQTTPasswordHash: passwordHash,
	})
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateHardwareID) || errors.Is(err, repositories.ErrProvisioningTokenInvalid) {
			service.logger.Warn("audit", "event", "device.registration_rejected", "reason", err.Error())
		}
		return nil, err
	}

	service.logger.Info("audit", "event", "device.provisioned", "deviceId", deviceID)

	return &DeviceRegistration{Device: device, MQTTUsername: deviceID, MQTTPassword: password}, nil
}

// RotateCredentials issues a new MQTT password for an authorized caller; the old password stops working immediately.
func (service *DeviceProvisioningService) RotateCredentials(ctx context.Context, deviceID string) (*DeviceRegistration, error) {
	if err := validateDeviceID(deviceID); err != nil {
		return nil, err
	}

	password, err := randomSecret("", 32)
	if err != nil {
		return nil, err
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash mqtt password: %w", err)
	}

	if err := service.repository.SetCredentials(ctx, deviceID, passwordHash, service.now().UTC()); err != nil {
		return nil, err
	}

	service.logger.Info("audit", "event", "device.credentials_rotated", "deviceId", deviceID)

	return &DeviceRegistration{MQTTUsername: deviceID, MQTTPassword: password}, nil
}

func (service *DeviceProvisioningService) SetLifecycleStatus(ctx context.Context, deviceID string, status string) error {
	if err := validateDeviceID(deviceID); err != nil {
		return err
	}
	switch status {
	case models.LifecycleActive, models.LifecycleDisabled, models.LifecycleRevoked:
	default:
		return errors.Join(ErrDeviceValidation, ErrInvalidLifecycle)
	}

	if err := service.repository.SetLifecycleStatus(ctx, deviceID, status); err != nil {
		return err
	}

	service.logger.Info("audit", "event", "device.lifecycle_changed", "deviceId", deviceID, "status", status)
	return nil
}
