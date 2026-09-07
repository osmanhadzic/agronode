package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
)

var ErrDeviceValidation = errors.New("device validation failed")

type DeviceService struct {
	repository     repositories.DeviceRepository
	logger         *slog.Logger
	eventPublisher DeviceEventPublisher
}

type DevicePresenceUpdater interface {
	UpdatePresence(ctx context.Context, deviceID string, seenAt time.Time) error
}

type DeviceEventPublisher interface {
	PublishDeviceEvent(event models.DeviceStatusEvent)
}

type DeviceSensorDiscoveryUpdater interface {
	UpdateDiscoveredSensors(ctx context.Context, deviceID string, sensorNames []string) error
}

const (
	defaultDeviceListPage  = 1
	defaultDeviceListLimit = 20
	maxDeviceListLimit     = 100
)

type DeviceListParams struct {
	Page   int
	Limit  int
	Status string
	Search string
	Tags   []string
}

func NewDeviceService(repository repositories.DeviceRepository, logger *slog.Logger) *DeviceService {
	return &DeviceService{
		repository: repository,
		logger:     logger,
	}
}

// SetEventPublisher sets the event publisher for device status events
func (service *DeviceService) SetEventPublisher(publisher DeviceEventPublisher) {
	service.eventPublisher = publisher
}

// RegisterDevice registers a new device or returns existing one (idempotent)
func (service *DeviceService) RegisterDevice(ctx context.Context, deviceID string, firmwareVersion string, metadata models.DeviceMetadata, apiKey string, provisioningToken string, tags []string) (*models.Device, error) {
	if err := validateDeviceID(deviceID); err != nil {
		return nil, err
	}

	if err := validateDeviceMetadata(metadata); err != nil {
		return nil, err
	}

	normalizedFirmware := strings.TrimSpace(firmwareVersion)
	normalizedTags := normalizeTags(tags)

	existingDevice, err := service.repository.GetByDeviceID(ctx, deviceID)
	if err != nil && !errors.Is(err, repositories.ErrDeviceNotFound) {
		return nil, err
	}

	if err == nil && existingDevice != nil {
		if tags != nil {
			existingDevice.Tags = normalizedTags
		}

		previousFirmware := existingDevice.FirmwareVersion
		if normalizedFirmware != "" && normalizedFirmware != previousFirmware {
			existingDevice.FirmwareVersion = normalizedFirmware
			if updateErr := service.repository.Update(ctx, existingDevice); updateErr != nil {
				return nil, updateErr
			}

			service.logAudit(
				"device.firmware_updated",
				"deviceId", deviceID,
				"oldFirmwareVersion", previousFirmware,
				"newFirmwareVersion", normalizedFirmware,
			)
		}

		service.logAudit(
			"device.registration",
			"deviceId", deviceID,
			"newRegistration", false,
			"firmwareVersion", existingDevice.FirmwareVersion,
		)

		return existingDevice, nil
	}

	device := &models.Device{
		DeviceID:              deviceID,
		Status:                models.DeviceStatusUnknown,
		FirmwareVersion:       normalizedFirmware,
		Metadata:              metadata,
		Tags:                  normalizedTags,
		DesiredState:          models.DeviceShadowState{},
		ReportedState:         models.DeviceShadowState{},
		DiscoveredSensors:     []string{},
		APIKeyHash:            hashDeviceSecret(apiKey),
		ProvisioningTokenHash: hashDeviceSecret(provisioningToken),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := service.repository.Create(ctx, device); err != nil {
		return nil, err
	}

	service.logAudit(
		"device.registration",
		"deviceId", deviceID,
		"newRegistration", true,
		"firmwareVersion", normalizedFirmware,
	)

	return device, nil
}

// GetDevice retrieves a device by its device_id
func (service *DeviceService) GetDevice(ctx context.Context, deviceID string) (*models.Device, error) {
	if err := validateDeviceID(deviceID); err != nil {
		return nil, err
	}

	return service.repository.GetByDeviceID(ctx, deviceID)
}

// ListDevices retrieves devices using pagination, filtering and search
func (service *DeviceService) ListDevices(ctx context.Context, params DeviceListParams) ([]models.Device, error) {
	normalizedParams, err := normalizeDeviceListParams(params)
	if err != nil {
		return nil, err
	}

	return service.repository.List(ctx, repositories.DeviceListQuery{
		Page:   normalizedParams.Page,
		Limit:  normalizedParams.Limit,
		Status: normalizedParams.Status,
		Search: normalizedParams.Search,
		Tags:   normalizedParams.Tags,
	})
}

// UpdatePresence updates device presence based on telemetry activity.
func (service *DeviceService) UpdatePresence(ctx context.Context, deviceID string, seenAt time.Time) error {
	if err := validateDeviceID(deviceID); err != nil {
		return err
	}

	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}

	device, err := service.repository.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return err
	}

	oldStatus := device.Status
	device.Status = models.DeviceStatusOnline
	device.LastSeen = &seenAt

	if err := service.repository.Update(ctx, device); err != nil {
		return err
	}

	if oldStatus != models.DeviceStatusOnline && service.eventPublisher != nil {
		service.eventPublisher.PublishDeviceEvent(models.DeviceStatusEvent{
			DeviceID:  deviceID,
			OldStatus: oldStatus,
			NewStatus: models.DeviceStatusOnline,
			EventType: models.EventDeviceOnline,
			Timestamp: time.Now().UTC(),
		})
	}

	if oldStatus != models.DeviceStatusOnline {
		service.logAudit(
			"device.status_changed",
			"deviceId", deviceID,
			"oldStatus", oldStatus,
			"newStatus", models.DeviceStatusOnline,
			"source", "telemetry",
		)
	}

	return nil
}

// MarkInactiveDevicesOffline marks online devices as offline if they haven't been seen since threshold
func (service *DeviceService) MarkInactiveDevicesOffline(ctx context.Context, inactivityThreshold time.Duration) (int, error) {
	if inactivityThreshold <= 0 {
		return 0, errors.Join(ErrDeviceValidation, errors.New("inactivity threshold must be positive"))
	}

	staleTime := time.Now().UTC().Add(-inactivityThreshold)

	inactiveDevices, err := service.repository.ListInactiveOnline(ctx, staleTime)
	if err != nil {
		return 0, err
	}

	if len(inactiveDevices) == 0 {
		return 0, nil
	}

	marked := 0
	for i := range inactiveDevices {
		oldStatus := inactiveDevices[i].Status
		inactiveDevices[i].Status = models.DeviceStatusOffline

		if err := service.repository.Update(ctx, &inactiveDevices[i]); err != nil {
			if service.logger != nil {
				service.logger.Warn("device offline marking failed", "deviceId", inactiveDevices[i].DeviceID, "error", err)
			}
			continue
		}

		marked++

		if oldStatus == models.DeviceStatusOnline && service.eventPublisher != nil {
			service.eventPublisher.PublishDeviceEvent(models.DeviceStatusEvent{
				DeviceID:  inactiveDevices[i].DeviceID,
				OldStatus: oldStatus,
				NewStatus: models.DeviceStatusOffline,
				EventType: models.EventDeviceOffline,
				Timestamp: time.Now().UTC(),
			})
		}

		if oldStatus == models.DeviceStatusOnline {
			service.logAudit(
				"device.status_changed",
				"deviceId", inactiveDevices[i].DeviceID,
				"oldStatus", oldStatus,
				"newStatus", models.DeviceStatusOffline,
				"source", "offline_detection_worker",
			)
		}

		if service.logger != nil {
			service.logger.Info("device marked offline", "deviceId", inactiveDevices[i].DeviceID, "lastSeen", inactiveDevices[i].LastSeen)
		}
	}

	return marked, nil
}

// UpdateDiscoveredSensors merges newly observed sensors into the device registry.
func (service *DeviceService) UpdateDiscoveredSensors(ctx context.Context, deviceID string, sensorNames []string) error {
	if err := validateDeviceID(deviceID); err != nil {
		return err
	}

	normalizedSensors := normalizeSensorNames(sensorNames)
	if len(normalizedSensors) == 0 {
		return nil
	}

	device, err := service.repository.GetByDeviceID(ctx, deviceID)
	if err != nil {
		return err
	}

	existing := make(map[string]struct{}, len(device.DiscoveredSensors))
	for _, sensor := range device.DiscoveredSensors {
		existing[sensor] = struct{}{}
	}

	changed := false
	for _, sensor := range normalizedSensors {
		if _, ok := existing[sensor]; ok {
			continue
		}
		device.DiscoveredSensors = append(device.DiscoveredSensors, sensor)
		existing[sensor] = struct{}{}
		changed = true
	}

	if !changed {
		return nil
	}

	sort.Strings(device.DiscoveredSensors)
	return service.repository.Update(ctx, device)
}

// validateDeviceID validates device ID format
func validateDeviceID(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.Join(ErrDeviceValidation, errors.New("deviceId is required"))
	}

	if len(deviceID) > 255 {
		return errors.Join(ErrDeviceValidation, errors.New("deviceId too long"))
	}

	return nil
}

func validateDeviceMetadata(metadata models.DeviceMetadata) error {
	if metadata.Battery != nil && (*metadata.Battery < 0 || *metadata.Battery > 100) {
		return errors.Join(ErrDeviceValidation, errors.New("battery must be between 0 and 100"))
	}

	if metadata.SignalStrength != nil && (*metadata.SignalStrength < -200 || *metadata.SignalStrength > 0) {
		return errors.Join(ErrDeviceValidation, errors.New("signalStrength must be between -200 and 0"))
	}

	return nil
}

func hashDeviceSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func (service *DeviceService) logAudit(event string, attributes ...any) {
	if service.logger == nil {
		return
	}

	payload := append([]any{"event", event}, attributes...)
	service.logger.Info("device audit", payload...)
}

func normalizeSensorNames(sensorNames []string) []string {
	unique := make(map[string]struct{}, len(sensorNames))
	for _, sensorName := range sensorNames {
		sensorName = strings.TrimSpace(sensorName)
		if sensorName == "" {
			continue
		}
		unique[sensorName] = struct{}{}
	}

	normalized := make([]string, 0, len(unique))
	for sensorName := range unique {
		normalized = append(normalized, sensorName)
	}

	sort.Strings(normalized)
	return normalized
}

func normalizeDeviceListParams(params DeviceListParams) (DeviceListParams, error) {
	normalized := DeviceListParams{
		Page:   params.Page,
		Limit:  params.Limit,
		Status: strings.TrimSpace(params.Status),
		Search: strings.TrimSpace(params.Search),
		Tags:   normalizeTags(params.Tags),
	}

	if normalized.Page == 0 {
		normalized.Page = defaultDeviceListPage
	}

	if normalized.Limit == 0 {
		normalized.Limit = defaultDeviceListLimit
	}

	if normalized.Page < 1 {
		return DeviceListParams{}, errors.Join(ErrDeviceValidation, errors.New("page must be >= 1"))
	}

	if normalized.Limit < 1 || normalized.Limit > maxDeviceListLimit {
		return DeviceListParams{}, errors.Join(ErrDeviceValidation, errors.New("limit must be between 1 and 100"))
	}

	if normalized.Status != "" &&
		normalized.Status != models.DeviceStatusUnknown &&
		normalized.Status != models.DeviceStatusOnline &&
		normalized.Status != models.DeviceStatusOffline {
		return DeviceListParams{}, errors.Join(ErrDeviceValidation, errors.New("status must be one of unknown, online, offline"))
	}

	if len(normalized.Search) > 255 {
		return DeviceListParams{}, errors.Join(ErrDeviceValidation, errors.New("search too long"))
	}

	return normalized, nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		unique[tag] = struct{}{}
	}

	if len(unique) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(unique))
	for tag := range unique {
		normalized = append(normalized, tag)
	}

	sort.Strings(normalized)
	return normalized
}
