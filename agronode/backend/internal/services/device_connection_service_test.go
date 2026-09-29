package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"agronode/backend/internal/models"
)

func TestDeviceConnectionLifecycle(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		status     string
		serial     string
		expiresAt  *time.Time
		wantStatus string
		wantErr    error
	}{
		{name: "connect", action: "connect", status: models.DeviceStatusOffline, serial: "SERIAL-1", wantStatus: models.DeviceStatusOnline},
		{name: "heartbeat", action: "heartbeat", status: models.DeviceStatusOnline, serial: "SERIAL-1", wantStatus: models.DeviceStatusOnline},
		{name: "disconnect", action: "disconnect", status: models.DeviceStatusOnline, serial: "SERIAL-1", wantStatus: models.DeviceStatusOffline},
		{name: "wrong key", action: "connect", status: models.DeviceStatusOffline, wantErr: ErrDeviceAuthentication},
		{name: "unprovisioned", action: "connect", status: models.DeviceStatusOffline, wantErr: ErrDeviceNotProvisioned},
		{name: "inactive", action: "connect", status: models.DeviceStatusOffline, wantErr: ErrDeviceInactive},
		{name: "missing certificate", action: "connect", status: models.DeviceStatusOffline, wantErr: ErrDeviceCertificateMismatch},
		{name: "certificate mismatch", action: "connect", status: models.DeviceStatusOffline, serial: "SERIAL-2", wantErr: ErrDeviceCertificateMismatch},
		{name: "expired certificate", action: "connect", status: models.DeviceStatusOffline, serial: "SERIAL-1", expiresAt: timePtr(time.Now().UTC().Add(-time.Second)), wantErr: ErrDeviceCertificateExpired},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provisioningStatus := models.ProvisioningStatusProvisioned
			registrationStatus := models.RegistrationStatusActive
			apiKey := "secret-api-key"
			if test.name == "wrong key" {
				apiKey = "incorrect-key"
			}
			if test.name == "unprovisioned" {
				provisioningStatus = models.ProvisioningStatusPending
			}
			if test.name == "inactive" {
				registrationStatus = models.RegistrationStatusInactive
			}

			device := &models.Device{
				DeviceID:             "esp32-test",
				Status:               test.status,
				RegistrationStatus:   registrationStatus,
				ProvisioningStatus:   provisioningStatus,
				APIKeyHash:           hashDeviceSecret("secret-api-key"),
				CertificateSerial:    "SERIAL-1",
				CertificateExpiresAt: test.expiresAt,
			}
			repository := &deviceRepositoryStub{
				getByDeviceIDResult: device,
			}
			service := NewDeviceService(repository, nil)

			var result *models.Device
			var err error
			switch test.action {
			case "connect":
				result, err = service.Connect(context.Background(), "esp32-test", apiKey, test.serial)
			case "heartbeat":
				result, err = service.Heartbeat(context.Background(), "esp32-test", apiKey, test.serial)
			case "disconnect":
				result, err = service.Disconnect(context.Background(), "esp32-test", apiKey, test.serial)
			}

			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error %v, got %v", test.wantErr, err)
				}
				if repository.updateInput != nil {
					t.Fatal("did not expect device update after failed authentication")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result == nil || result.Status != test.wantStatus {
				t.Fatalf("expected status %q, got %#v", test.wantStatus, result)
			}
			if repository.updateInput == nil || repository.updateInput.LastSeen == nil {
				t.Fatal("expected the authenticated connection to update lastSeen")
			}
			if time.Since(*repository.updateInput.LastSeen) > time.Minute {
				t.Fatalf("expected a recent lastSeen timestamp, got %s", repository.updateInput.LastSeen)
			}
		})
	}
}

func TestDeviceConnectionAllowsDevicesWithoutConfiguredCertificate(t *testing.T) {
	device := &models.Device{
		DeviceID:           "esp32-api-key-only",
		RegistrationStatus: models.RegistrationStatusActive,
		ProvisioningStatus: models.ProvisioningStatusProvisioned,
		APIKeyHash:         hashDeviceSecret("secret-api-key"),
	}
	repository := &deviceRepositoryStub{getByDeviceIDResult: device}
	service := NewDeviceService(repository, nil)

	if _, err := service.Connect(context.Background(), device.DeviceID, "secret-api-key", ""); err != nil {
		t.Fatalf("expected API-key-only connection to succeed when no certificate is registered, got %v", err)
	}
}

func TestDeviceConnectionRejectsExpiredCertificateAtExpiryTime(t *testing.T) {
	expiresAt := time.Now().UTC()
	device := &models.Device{
		DeviceID:             "esp32-expired",
		RegistrationStatus:   models.RegistrationStatusActive,
		ProvisioningStatus:   models.ProvisioningStatusProvisioned,
		APIKeyHash:           hashDeviceSecret("secret-api-key"),
		CertificateSerial:    "SERIAL-1",
		CertificateExpiresAt: &expiresAt,
	}
	service := NewDeviceService(&deviceRepositoryStub{getByDeviceIDResult: device}, nil)

	if _, err := service.Connect(context.Background(), device.DeviceID, "secret-api-key", "SERIAL-1"); !errors.Is(err, ErrDeviceCertificateExpired) {
		t.Fatalf("expected expired-certificate error, got %v", err)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
