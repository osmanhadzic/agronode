package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type deviceRegistrationServiceStub struct {
	getDeviceResult   *models.Device
	getDeviceError    error
	requestedID       string
	listDevicesInput  services.DeviceListParams
	listDevicesResult []models.Device
	listDevicesError  error
	registerInput     *struct {
		deviceID          string
		firmware          string
		metadata          models.DeviceMetadata
		apiKey            string
		provisioningToken string
		tags              []string
	}
	registerResult *models.Device
	registerError  error
}

func (stub *deviceRegistrationServiceStub) RegisterDevice(_ context.Context, deviceID string, firmwareVersion string, metadata models.DeviceMetadata, apiKey string, provisioningToken string, tags []string) (*models.Device, error) {
	stub.registerInput = &struct {
		deviceID          string
		firmware          string
		metadata          models.DeviceMetadata
		apiKey            string
		provisioningToken string
		tags              []string
	}{deviceID: deviceID, firmware: firmwareVersion, metadata: metadata, apiKey: apiKey, provisioningToken: provisioningToken, tags: tags}
	if stub.registerError != nil {
		return nil, stub.registerError
	}
	return stub.registerResult, nil
}

func (stub *deviceRegistrationServiceStub) GetDevice(_ context.Context, deviceID string) (*models.Device, error) {
	stub.requestedID = deviceID
	if stub.getDeviceError != nil {
		return nil, stub.getDeviceError
	}
	return stub.getDeviceResult, nil
}

func (stub *deviceRegistrationServiceStub) ListDevices(_ context.Context, params services.DeviceListParams) ([]models.Device, error) {
	stub.listDevicesInput = params
	if stub.listDevicesError != nil {
		return nil, stub.listDevicesError
	}
	return stub.listDevicesResult, nil
}

func testDeviceLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns device details", func(t *testing.T) {
		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		service := &deviceRegistrationServiceStub{
			getDeviceResult: &models.Device{
				ID:              7,
				DeviceID:        "esp32-lab",
				Status:          models.DeviceStatusOnline,
				FirmwareVersion: "v1.2.3",
				Metadata:        models.DeviceMetadata{Hardware: map[string]string{"model": "ESP32", "board": "devkit"}},
				CreatedAt:       now,
				UpdatedAt:       now,
			},
		}

		router := gin.New()
		api := router.Group("/api")
		RegisterDeviceRoutes(api, testDeviceLogger(), service)

		request := httptest.NewRequest(http.MethodGet, "/api/devices/esp32-lab", nil)
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
		}

		if service.requestedID != "esp32-lab" {
			t.Fatalf("expected requested device id %q, got %q", "esp32-lab", service.requestedID)
		}

		var responseBody map[string]any
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if responseBody["deviceId"] != "esp32-lab" {
			t.Fatalf("expected deviceId %q, got %#v", "esp32-lab", responseBody["deviceId"])
		}

		if responseBody["status"] != models.DeviceStatusOnline {
			t.Fatalf("expected status %q, got %#v", models.DeviceStatusOnline, responseBody["status"])
		}

		if _, ok := responseBody["metadata"]; !ok {
			t.Fatal("expected metadata field in response")
		}
	})

	t.Run("returns 404 when device does not exist", func(t *testing.T) {
		service := &deviceRegistrationServiceStub{getDeviceError: repositories.ErrDeviceNotFound}

		router := gin.New()
		api := router.Group("/api")
		RegisterDeviceRoutes(api, testDeviceLogger(), service)

		request := httptest.NewRequest(http.MethodGet, "/api/devices/missing-device", nil)
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, responseRecorder.Code)
		}

		var responseBody map[string]any
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if responseBody["error"] != "device not found" {
			t.Fatalf("expected error %q, got %#v", "device not found", responseBody["error"])
		}
	})
}

func TestRegisterDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("accepts metadata and returns created device", func(t *testing.T) {
		battery := 87.5
		signal := -61.0
		service := &deviceRegistrationServiceStub{
			registerResult: &models.Device{
				ID:              11,
				DeviceID:        "esp32-lab",
				Status:          models.DeviceStatusUnknown,
				FirmwareVersion: "v1.2.3",
				Metadata: models.DeviceMetadata{
					Battery:        &battery,
					SignalStrength: &signal,
					Hardware:       map[string]string{"model": "ESP32", "rev": "1"},
				},
				CreatedAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			},
		}

		router := gin.New()
		api := router.Group("/api")
		RegisterDeviceRoutes(api, testDeviceLogger(), service)

		body := `{"deviceId":"esp32-lab","firmwareVersion":"v1.2.3","metadata":{"battery":87.5,"signalStrength":-61,"hardware":{"model":"ESP32","rev":"1"}},"apiKey":"api-key-value","provisioningToken":"provisioning-token-value","tags":["greenhouse","lab"]}`
		request := httptest.NewRequest(http.MethodPost, "/api/devices/register", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
		}

		if service.registerInput == nil || service.registerInput.deviceID != "esp32-lab" {
			t.Fatal("expected register input to be captured")
		}

		if service.registerInput.apiKey != "api-key-value" {
			t.Fatalf("expected api key to be forwarded, got %q", service.registerInput.apiKey)
		}

		if service.registerInput.provisioningToken != "provisioning-token-value" {
			t.Fatalf("expected provisioning token to be forwarded, got %q", service.registerInput.provisioningToken)
		}

		if len(service.registerInput.tags) != 2 || service.registerInput.tags[0] != "greenhouse" || service.registerInput.tags[1] != "lab" {
			t.Fatalf("expected tags [greenhouse lab], got %v", service.registerInput.tags)
		}

		if service.registerInput.metadata.Battery == nil || *service.registerInput.metadata.Battery != 87.5 {
			t.Fatalf("expected battery metadata 87.5, got %#v", service.registerInput.metadata.Battery)
		}

		if service.registerInput.metadata.SignalStrength == nil || *service.registerInput.metadata.SignalStrength != -61 {
			t.Fatalf("expected signal strength metadata -61, got %#v", service.registerInput.metadata.SignalStrength)
		}

		var responseBody map[string]any
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if _, ok := responseBody["metadata"]; !ok {
			t.Fatal("expected metadata field in register response")
		}
	})
}

func TestListDevices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("passes pagination filter and search params", func(t *testing.T) {
		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		service := &deviceRegistrationServiceStub{
			listDevicesResult: []models.Device{
				{
					ID:        3,
					DeviceID:  "esp32-lab",
					Status:    models.DeviceStatusOnline,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		router := gin.New()
		api := router.Group("/api")
		RegisterDeviceRoutes(api, testDeviceLogger(), service)

		request := httptest.NewRequest(http.MethodGet, "/api/devices?page=2&limit=5&status=online&search=lab&tags=greenhouse,lab", nil)
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
		}

		if service.listDevicesInput.Page != 2 {
			t.Fatalf("expected page 2, got %d", service.listDevicesInput.Page)
		}

		if service.listDevicesInput.Limit != 5 {
			t.Fatalf("expected limit 5, got %d", service.listDevicesInput.Limit)
		}

		if service.listDevicesInput.Status != models.DeviceStatusOnline {
			t.Fatalf("expected status %q, got %q", models.DeviceStatusOnline, service.listDevicesInput.Status)
		}

		if service.listDevicesInput.Search != "lab" {
			t.Fatalf("expected search %q, got %q", "lab", service.listDevicesInput.Search)
		}

		if len(service.listDevicesInput.Tags) != 2 || service.listDevicesInput.Tags[0] != "greenhouse" || service.listDevicesInput.Tags[1] != "lab" {
			t.Fatalf("expected tags [greenhouse lab], got %v", service.listDevicesInput.Tags)
		}

		var responseBody []map[string]any
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(responseBody) != 1 {
			t.Fatalf("expected 1 device in response, got %d", len(responseBody))
		}
	})

	t.Run("returns 400 for non-numeric page", func(t *testing.T) {
		service := &deviceRegistrationServiceStub{}

		router := gin.New()
		api := router.Group("/api")
		RegisterDeviceRoutes(api, testDeviceLogger(), service)

		request := httptest.NewRequest(http.MethodGet, "/api/devices?page=abc", nil)
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, responseRecorder.Code)
		}
	})

	t.Run("returns 400 for invalid status", func(t *testing.T) {
		service := &deviceRegistrationServiceStub{
			listDevicesError: errors.Join(services.ErrDeviceValidation, errors.New("status must be one of unknown, online, offline")),
		}

		router := gin.New()
		api := router.Group("/api")
		RegisterDeviceRoutes(api, testDeviceLogger(), service)

		request := httptest.NewRequest(http.MethodGet, "/api/devices?status=invalid", nil)
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		if responseRecorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, responseRecorder.Code)
		}
	})
}
