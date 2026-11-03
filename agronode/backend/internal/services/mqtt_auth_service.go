package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"strings"

	"agronode/backend/internal/auth"
	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
)

var ErrMQTTNotAuthorized = errors.New("not authorized")

const mqttTopicRoot = "agronode"

// dummyPasswordHash equalizes timing for unknown usernames.
var dummyPasswordHash, _ = auth.HashPassword("agronode-dummy-password")

type MQTTCredentialLookup interface {
	GetByMQTTUsername(ctx context.Context, username string) (*models.Device, error)
}

// MQTTAuthService decides broker authentication and topic authorization from server-side records only.
// Nothing is cached so that disabling or revoking a device takes effect on the next broker call.
type MQTTAuthService struct {
	devices         MQTTCredentialLookup
	serviceUsername string
	servicePassword string
	logger          *slog.Logger
}

func NewMQTTAuthService(devices MQTTCredentialLookup, serviceUsername string, servicePassword string, logger *slog.Logger) *MQTTAuthService {
	return &MQTTAuthService{devices: devices, serviceUsername: serviceUsername, servicePassword: servicePassword, logger: logger}
}

func (service *MQTTAuthService) isServiceUser(username string) bool {
	return service.serviceUsername != "" && service.servicePassword != "" &&
		subtle.ConstantTimeCompare([]byte(username), []byte(service.serviceUsername)) == 1
}

// Authenticate returns nil only for the backend service account or an active device with a matching password.
func (service *MQTTAuthService) Authenticate(ctx context.Context, username string, clientID string, password string) error {
	if username == "" || password == "" {
		return ErrMQTTNotAuthorized
	}

	if service.isServiceUser(username) {
		if subtle.ConstantTimeCompare([]byte(password), []byte(service.servicePassword)) == 1 {
			return nil
		}
		return ErrMQTTNotAuthorized
	}

	// The client id is bound to the identity so one device cannot take over another's session.
	if clientID != username {
		return ErrMQTTNotAuthorized
	}

	device, err := service.devices.GetByMQTTUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repositories.ErrDeviceNotFound) {
			_ = auth.ComparePassword(dummyPasswordHash, password)
			return ErrMQTTNotAuthorized
		}
		return err
	}

	hash := device.MQTTPasswordHash
	if hash == "" {
		hash = dummyPasswordHash
	}
	passwordErr := auth.ComparePassword(hash, password)
	if passwordErr != nil || device.MQTTPasswordHash == "" || device.LifecycleStatus != models.LifecycleActive {
		return ErrMQTTNotAuthorized
	}

	return nil
}

// activeDevice re-reads the device so the decision reflects the current lifecycle state.
func (service *MQTTAuthService) activeDevice(ctx context.Context, username string) (*models.Device, error) {
	device, err := service.devices.GetByMQTTUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repositories.ErrDeviceNotFound) {
			return nil, ErrMQTTNotAuthorized
		}
		return nil, err
	}
	if device.LifecycleStatus != models.LifecycleActive || device.MQTTUsername == nil || *device.MQTTUsername != username {
		return nil, ErrMQTTNotAuthorized
	}
	return device, nil
}

func splitTopic(topic string) ([]string, bool) {
	if topic == "" || strings.ContainsAny(topic, "+#\x00") {
		return nil, false
	}
	return strings.Split(topic, "/"), true
}

// AuthorizePublish: devices may publish only agronode/{ownDeviceId}/telemetry and /status.
func (service *MQTTAuthService) AuthorizePublish(ctx context.Context, username string, topic string) error {
	if service.isServiceUser(username) {
		if strings.HasPrefix(topic, mqttTopicRoot+"/") && !strings.ContainsAny(topic, "+#") {
			return nil
		}
		return ErrMQTTNotAuthorized
	}

	parts, ok := splitTopic(topic)
	if !ok || len(parts) != 3 || parts[0] != mqttTopicRoot || (parts[2] != "telemetry" && parts[2] != "status") {
		return ErrMQTTNotAuthorized
	}

	device, err := service.activeDevice(ctx, username)
	if err != nil {
		return err
	}
	if parts[1] != device.DeviceID {
		return ErrMQTTNotAuthorized
	}

	return nil
}

// AuthorizeSubscribe: devices may subscribe only to agronode/{ownDeviceId}/activation, without wildcards.
func (service *MQTTAuthService) AuthorizeSubscribe(ctx context.Context, username string, topics []string) error {
	if len(topics) == 0 {
		return ErrMQTTNotAuthorized
	}

	if service.isServiceUser(username) {
		for _, topic := range topics {
			if topic != mqttTopicRoot+"/#" && !strings.HasPrefix(topic, mqttTopicRoot+"/") {
				return ErrMQTTNotAuthorized
			}
		}
		return nil
	}

	device, err := service.activeDevice(ctx, username)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		parts, ok := splitTopic(topic)
		if !ok || len(parts) != 3 || parts[0] != mqttTopicRoot || parts[1] != device.DeviceID || parts[2] != "activation" {
			return ErrMQTTNotAuthorized
		}
	}

	return nil
}
