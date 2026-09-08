package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
)

type deviceRepositoryStub struct {
	created        []*models.Device
	listed         []models.Device
	listQueryInput repositories.DeviceListQuery

	getByDeviceIDResult *models.Device
	getByDeviceIDError  error

	updateInput *models.Device
	updateError error
}

type deviceEventPublisherStub struct {
	publishedEvents []models.DeviceStatusEvent
}

func (stub *deviceEventPublisherStub) PublishDeviceEvent(event models.DeviceStatusEvent) {
	stub.publishedEvents = append(stub.publishedEvents, event)
}

type deviceSensorDiscoveryUpdaterStub struct {
	updatedDeviceID string
	updatedSensors  []string
	err             error
}

func (stub *deviceSensorDiscoveryUpdaterStub) UpdateDiscoveredSensors(_ context.Context, deviceID string, sensorNames []string) error {
	stub.updatedDeviceID = deviceID
	stub.updatedSensors = append([]string{}, sensorNames...)
	return stub.err
}

func (repository *deviceRepositoryStub) Create(_ context.Context, device *models.Device) error {
	repository.created = append(repository.created, device)
	return nil
}

func (repository *deviceRepositoryStub) GetByDeviceID(context.Context, string) (*models.Device, error) {
	if repository.getByDeviceIDError != nil {
		return nil, repository.getByDeviceIDError
	}
	return repository.getByDeviceIDResult, nil
}

func (repository *deviceRepositoryStub) Update(_ context.Context, device *models.Device) error {
	repository.updateInput = device
	if repository.updateError != nil {
		return repository.updateError
	}
	return nil
}

func (repository *deviceRepositoryStub) List(_ context.Context, query repositories.DeviceListQuery) ([]models.Device, error) {
	repository.listQueryInput = query
	return repository.listed, nil
}

func (repository *deviceRepositoryStub) ListInactiveOnline(context.Context, time.Time) ([]models.Device, error) {
	return repository.listed, nil
}

func TestDeviceService_UpdatePresence(t *testing.T) {
	t.Run("marks device online and updates lastSeen", func(t *testing.T) {
		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		repository := &deviceRepositoryStub{
			getByDeviceIDResult: &models.Device{DeviceID: "esp32-lab", Status: models.DeviceStatusUnknown},
		}
		service := NewDeviceService(repository, nil)

		err := service.UpdatePresence(context.Background(), "esp32-lab", now)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if repository.updateInput == nil {
			t.Fatal("expected repository.Update to be called")
		}

		if repository.updateInput.Status != models.DeviceStatusOnline {
			t.Fatalf("expected status %q, got %q", models.DeviceStatusOnline, repository.updateInput.Status)
		}

		if repository.updateInput.LastSeen == nil || !repository.updateInput.LastSeen.Equal(now) {
			t.Fatalf("expected lastSeen to equal %s", now)
		}
	})

	t.Run("returns validation error for empty device id", func(t *testing.T) {
		repository := &deviceRepositoryStub{}
		service := NewDeviceService(repository, nil)

		err := service.UpdatePresence(context.Background(), "   ", time.Now().UTC())
		if !errors.Is(err, ErrDeviceValidation) {
			t.Fatalf("expected ErrDeviceValidation, got %v", err)
		}
	})
}

func TestDeviceService_MarkInactiveDevicesOffline(t *testing.T) {
	t.Run("marks inactive online devices as offline", func(t *testing.T) {
		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		fifteenMinutesAgo := now.Add(-15 * time.Minute)

		repository := &deviceRepositoryStub{
			listed: []models.Device{
				{
					ID:       1,
					DeviceID: "device1",
					Status:   models.DeviceStatusOnline,
					LastSeen: &fifteenMinutesAgo,
				},
				{
					ID:       2,
					DeviceID: "device2",
					Status:   models.DeviceStatusOnline,
					LastSeen: &fifteenMinutesAgo,
				},
			},
		}

		service := NewDeviceService(repository, nil)

		marked, err := service.MarkInactiveDevicesOffline(context.Background(), 10*time.Minute)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if marked != 2 {
			t.Fatalf("expected 2 devices marked offline, got %d", marked)
		}

		if repository.updateInput == nil {
			t.Fatal("expected repository.Update to be called")
		}

		if repository.updateInput.Status != models.DeviceStatusOffline {
			t.Fatalf("expected status %q, got %q", models.DeviceStatusOffline, repository.updateInput.Status)
		}
	})

	t.Run("returns 0 when no inactive devices", func(t *testing.T) {
		repository := &deviceRepositoryStub{listed: []models.Device{}}
		service := NewDeviceService(repository, nil)

		marked, err := service.MarkInactiveDevicesOffline(context.Background(), 15*time.Minute)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if marked != 0 {
			t.Fatalf("expected 0 devices marked, got %d", marked)
		}
	})

	t.Run("returns validation error for negative threshold", func(t *testing.T) {
		repository := &deviceRepositoryStub{}
		service := NewDeviceService(repository, nil)

		_, err := service.MarkInactiveDevicesOffline(context.Background(), -1*time.Minute)
		if !errors.Is(err, ErrDeviceValidation) {
			t.Fatalf("expected ErrDeviceValidation, got %v", err)
		}
	})
}

func TestDeviceService_UpdatePresence_Events(t *testing.T) {
	t.Run("emits online event when transitioning to online", func(t *testing.T) {
		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		repository := &deviceRepositoryStub{
			getByDeviceIDResult: &models.Device{
				DeviceID: "esp32-lab",
				Status:   models.DeviceStatusUnknown,
			},
		}
		eventPublisher := &deviceEventPublisherStub{}
		service := NewDeviceService(repository, nil)
		service.SetEventPublisher(eventPublisher)

		err := service.UpdatePresence(context.Background(), "esp32-lab", now)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(eventPublisher.publishedEvents) != 1 {
			t.Fatalf("expected 1 event published, got %d", len(eventPublisher.publishedEvents))
		}

		event := eventPublisher.publishedEvents[0]
		if event.EventType != models.EventDeviceOnline {
			t.Fatalf("expected event type %q, got %q", models.EventDeviceOnline, event.EventType)
		}

		if event.OldStatus != models.DeviceStatusUnknown {
			t.Fatalf("expected old status %q, got %q", models.DeviceStatusUnknown, event.OldStatus)
		}

		if event.NewStatus != models.DeviceStatusOnline {
			t.Fatalf("expected new status %q, got %q", models.DeviceStatusOnline, event.NewStatus)
		}
	})

	t.Run("does not emit event when already online", func(t *testing.T) {
		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		repository := &deviceRepositoryStub{
			getByDeviceIDResult: &models.Device{
				DeviceID: "esp32-lab",
				Status:   models.DeviceStatusOnline,
				LastSeen: &now,
			},
		}
		eventPublisher := &deviceEventPublisherStub{}
		service := NewDeviceService(repository, nil)
		service.SetEventPublisher(eventPublisher)

		err := service.UpdatePresence(context.Background(), "esp32-lab", now)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(eventPublisher.publishedEvents) != 0 {
			t.Fatalf("expected no events published (duplicate prevention), got %d", len(eventPublisher.publishedEvents))
		}
	})
}

func TestDeviceService_MarkInactiveDevicesOffline_Events(t *testing.T) {
	t.Run("emits offline event for each marked device", func(t *testing.T) {
		fifteenMinutesAgo := time.Date(2026, 6, 1, 11, 45, 0, 0, time.UTC)
		repository := &deviceRepositoryStub{
			listed: []models.Device{
				{
					ID:       1,
					DeviceID: "device1",
					Status:   models.DeviceStatusOnline,
					LastSeen: &fifteenMinutesAgo,
				},
			},
		}
		eventPublisher := &deviceEventPublisherStub{}
		service := NewDeviceService(repository, nil)
		service.SetEventPublisher(eventPublisher)

		marked, err := service.MarkInactiveDevicesOffline(context.Background(), 10*time.Minute)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if marked != 1 {
			t.Fatalf("expected 1 device marked, got %d", marked)
		}

		if len(eventPublisher.publishedEvents) != 1 {
			t.Fatalf("expected 1 event published, got %d", len(eventPublisher.publishedEvents))
		}

		event := eventPublisher.publishedEvents[0]
		if event.EventType != models.EventDeviceOffline {
			t.Fatalf("expected event type %q, got %q", models.EventDeviceOffline, event.EventType)
		}

		if event.NewStatus != models.DeviceStatusOffline {
			t.Fatalf("expected new status %q, got %q", models.DeviceStatusOffline, event.NewStatus)
		}
	})
}

func TestDeviceService_RegisterDevice_Metadata(t *testing.T) {
	t.Run("preserves metadata on registration", func(t *testing.T) {
		battery := 91.5
		signal := -55.0
		repository := &deviceRepositoryStub{}
		service := NewDeviceService(repository, nil)

		device, err := service.RegisterDevice(context.Background(), "esp32-lab", "v1.2.3", models.DeviceMetadata{
			Battery:        &battery,
			SignalStrength: &signal,
			Hardware:       map[string]string{"model": "ESP32", "board": "devkit"},
		}, "", "", nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(repository.created) != 1 {
			t.Fatalf("expected 1 created device, got %d", len(repository.created))
		}

		if device == nil || device.Metadata.Battery == nil || *device.Metadata.Battery != 91.5 {
			t.Fatalf("expected metadata to be preserved, got %#v", device)
		}

		if device.DesiredState == nil {
			t.Fatal("expected desired state to be initialized")
		}

		if device.ReportedState == nil {
			t.Fatal("expected reported state to be initialized")
		}
	})

	t.Run("rejects invalid battery metadata", func(t *testing.T) {
		repository := &deviceRepositoryStub{}
		service := NewDeviceService(repository, nil)

		_, err := service.RegisterDevice(context.Background(), "esp32-lab", "v1.2.3", models.DeviceMetadata{Battery: floatPtr(120)}, "", "", nil)
		if !errors.Is(err, ErrDeviceValidation) {
			t.Fatalf("expected ErrDeviceValidation, got %v", err)
		}
	})
}

func TestDeviceService_RegisterDevice_AuthSecrets(t *testing.T) {
	t.Run("hashes api key and provisioning token before persistence", func(t *testing.T) {
		repository := &deviceRepositoryStub{}
		service := NewDeviceService(repository, nil)

		device, err := service.RegisterDevice(context.Background(), "esp32-lab", "v1.2.3", models.DeviceMetadata{}, "  api-key-value  ", "provisioning-token-value", nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if device == nil {
			t.Fatal("expected device result")
		}

		if device.APIKeyHash != hashDeviceSecret("api-key-value") {
			t.Fatalf("expected hashed api key, got %q", device.APIKeyHash)
		}

		if device.ProvisioningTokenHash != hashDeviceSecret("provisioning-token-value") {
			t.Fatalf("expected hashed provisioning token, got %q", device.ProvisioningTokenHash)
		}

		if len(repository.created) != 1 {
			t.Fatalf("expected 1 created device, got %d", len(repository.created))
		}

		if repository.created[0].APIKeyHash != device.APIKeyHash {
			t.Fatalf("expected persisted api key hash %q, got %q", device.APIKeyHash, repository.created[0].APIKeyHash)
		}
	})
}

func TestDeviceService_RegisterDevice_FirmwareUpdate(t *testing.T) {
	t.Run("updates firmware when device already exists", func(t *testing.T) {
		repository := &deviceRepositoryStub{
			getByDeviceIDResult: &models.Device{
				ID:              7,
				DeviceID:        "esp32-lab",
				Status:          models.DeviceStatusUnknown,
				FirmwareVersion: "v1.0.0",
			},
		}
		service := NewDeviceService(repository, nil)

		device, err := service.RegisterDevice(context.Background(), "esp32-lab", "v1.1.0", models.DeviceMetadata{}, "", "", nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(repository.created) != 0 {
			t.Fatalf("expected no create call for existing device, got %d", len(repository.created))
		}

		if repository.updateInput == nil {
			t.Fatal("expected update call for existing device firmware change")
		}

		if repository.updateInput.FirmwareVersion != "v1.1.0" {
			t.Fatalf("expected firmware version %q, got %q", "v1.1.0", repository.updateInput.FirmwareVersion)
		}

		if device == nil || device.FirmwareVersion != "v1.1.0" {
			t.Fatalf("expected returned firmware version %q, got %#v", "v1.1.0", device)
		}
	})
}

func TestDeviceService_ListDevices_Tags(t *testing.T) {
	t.Run("normalizes and forwards multiple tags", func(t *testing.T) {
		repository := &deviceRepositoryStub{}
		service := NewDeviceService(repository, nil)

		_, err := service.ListDevices(context.Background(), DeviceListParams{
			Page:  1,
			Limit: 20,
			Tags:  []string{" greenhouse ", "lab", "lab", ""},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(repository.listQueryInput.Tags) != 2 {
			t.Fatalf("expected 2 normalized tags, got %v", repository.listQueryInput.Tags)
		}

		if repository.listQueryInput.Tags[0] != "greenhouse" || repository.listQueryInput.Tags[1] != "lab" {
			t.Fatalf("expected normalized tags [greenhouse lab], got %v", repository.listQueryInput.Tags)
		}
	})
}

func TestDeviceService_UpdateDiscoveredSensors(t *testing.T) {
	t.Run("merges new sensors and deduplicates", func(t *testing.T) {
		repository := &deviceRepositoryStub{
			getByDeviceIDResult: &models.Device{
				DeviceID:          "esp32-lab",
				DiscoveredSensors: []string{"humidity"},
			},
		}
		service := NewDeviceService(repository, nil)

		err := service.UpdateDiscoveredSensors(context.Background(), "esp32-lab", []string{"temperature", "humidity", "co2", "temperature"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if repository.updateInput == nil {
			t.Fatal("expected repository.Update to be called")
		}

		if len(repository.updateInput.DiscoveredSensors) != 3 {
			t.Fatalf("expected 3 discovered sensors, got %v", repository.updateInput.DiscoveredSensors)
		}
	})

	t.Run("ignores empty sensor lists", func(t *testing.T) {
		repository := &deviceRepositoryStub{}
		service := NewDeviceService(repository, nil)

		err := service.UpdateDiscoveredSensors(context.Background(), "esp32-lab", []string{"   ", ""})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if repository.updateInput != nil {
			t.Fatal("expected no update for empty sensor list")
		}
	})
}

func floatPtr(value float64) *float64 {
	return &value
}
