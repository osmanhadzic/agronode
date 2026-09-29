package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
)

var (
	ErrDeviceNotProvisioned      = errors.New("device is not provisioned")
	ErrDeviceInactive            = errors.New("device registration is inactive")
	ErrDeviceAuthentication      = errors.New("device authentication failed")
	ErrDeviceCertificateMismatch = errors.New("device certificate does not match")
	ErrDeviceCertificateExpired  = errors.New("device certificate has expired")
)

// Connect authenticates a device and marks it online.
func (service *DeviceService) Connect(ctx context.Context, deviceID, apiKey, certificateSerial string) (*models.Device, error) {
	return service.updateConnection(ctx, deviceID, apiKey, certificateSerial, models.DeviceStatusOnline)
}

// Heartbeat authenticates a device and refreshes its online presence.
func (service *DeviceService) Heartbeat(ctx context.Context, deviceID, apiKey, certificateSerial string) (*models.Device, error) {
	return service.updateConnection(ctx, deviceID, apiKey, certificateSerial, models.DeviceStatusOnline)
}

// Disconnect authenticates a device before marking it offline.
func (service *DeviceService) Disconnect(ctx context.Context, deviceID, apiKey, certificateSerial string) (*models.Device, error) {
	return service.updateConnection(ctx, deviceID, apiKey, certificateSerial, models.DeviceStatusOffline)
}

func (service *DeviceService) updateConnection(ctx context.Context, deviceID, apiKey, certificateSerial, status string) (*models.Device, error) {
	if err := validateDeviceID(deviceID); err != nil {
		return nil, err
	}

	device, err := service.repository.GetByDeviceID(ctx, deviceID)
	if err != nil {
		if errors.Is(err, repositories.ErrDeviceNotFound) {
			return nil, ErrDeviceAuthentication
		}
		return nil, err
	}

	if device.RegistrationStatus != models.RegistrationStatusActive {
		return nil, ErrDeviceInactive
	}
	if device.ProvisioningStatus != models.ProvisioningStatusProvisioned {
		return nil, ErrDeviceNotProvisioned
	}
	providedKeyHash := hashDeviceSecret(apiKey)
	if device.APIKeyHash == "" || providedKeyHash == "" || subtle.ConstantTimeCompare([]byte(device.APIKeyHash), []byte(providedKeyHash)) != 1 {
		return nil, ErrDeviceAuthentication
	}

	if device.CertificateSerial != "" || device.CertificateExpiresAt != nil {
		if device.CertificateSerial == "" || strings.TrimSpace(certificateSerial) == "" || subtle.ConstantTimeCompare([]byte(device.CertificateSerial), []byte(certificateSerial)) != 1 {
			return nil, ErrDeviceCertificateMismatch
		}
	}

	now := time.Now().UTC()
	if device.CertificateExpiresAt != nil && !now.Before(*device.CertificateExpiresAt) {
		return nil, ErrDeviceCertificateExpired
	}

	oldStatus := device.Status
	device.Status = status
	device.LastSeen = &now
	if err := service.repository.Update(ctx, device); err != nil {
		return nil, err
	}

	if oldStatus != status {
		if service.eventPublisher != nil {
			eventType := models.EventDeviceOnline
			if status == models.DeviceStatusOffline {
				eventType = models.EventDeviceOffline
			}
			service.eventPublisher.PublishDeviceEvent(models.DeviceStatusEvent{
				DeviceID:  deviceID,
				OldStatus: oldStatus,
				NewStatus: status,
				EventType: eventType,
				Timestamp: now,
			})
		}
		service.logAudit("device.status_changed", "deviceId", deviceID, "oldStatus", oldStatus, "newStatus", status, "source", "device_connection")
	}

	return device, nil
}
