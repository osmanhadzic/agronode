package handlers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"agronode/backend/internal/models"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type deviceConnectionServiceStub struct {
	deviceID string
	apiKey   string
	serial   string
	result   *models.Device
	err      error
}

func (stub *deviceConnectionServiceStub) Connect(_ context.Context, deviceID, apiKey, serial string) (*models.Device, error) {
	return stub.call(deviceID, apiKey, serial)
}

func (stub *deviceConnectionServiceStub) Heartbeat(_ context.Context, deviceID, apiKey, serial string) (*models.Device, error) {
	return stub.call(deviceID, apiKey, serial)
}

func (stub *deviceConnectionServiceStub) Disconnect(_ context.Context, deviceID, apiKey, serial string) (*models.Device, error) {
	return stub.call(deviceID, apiKey, serial)
}

func (stub *deviceConnectionServiceStub) call(deviceID, apiKey, serial string) (*models.Device, error) {
	stub.deviceID = deviceID
	stub.apiKey = apiKey
	stub.serial = serial
	return stub.result, stub.err
}

func TestDeviceConnectionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("connect uses the API-key header and returns connection state", func(t *testing.T) {
		service := &deviceConnectionServiceStub{result: &models.Device{
			DeviceID: "esp32-lab",
			Status:   models.DeviceStatusOnline,
		}}
		router := gin.New()
		RegisterDeviceConnectionRoutes(router.Group("/api"), logger, service)

		request := httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/connect", nil)
		request.Header.Set(deviceAPIKeyHeader, "device-secret")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
		}
		if service.deviceID != "esp32-lab" || service.apiKey != "device-secret" {
			t.Fatalf("expected device credentials to be forwarded, got deviceID=%q apiKey=%q", service.deviceID, service.apiKey)
		}
	})

	t.Run("uses only a verified TLS client certificate serial", func(t *testing.T) {
		service := &deviceConnectionServiceStub{result: &models.Device{DeviceID: "esp32-lab", Status: models.DeviceStatusOnline}}
		router := gin.New()
		RegisterDeviceConnectionRoutes(router.Group("/api"), logger, service)

		request := httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/connect", nil)
		request.TLS = &tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{{SerialNumber: big.NewInt(1234)}},
			VerifiedChains:   [][]*x509.Certificate{{{SerialNumber: big.NewInt(1234)}}},
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
		}
		if service.serial != "1234" {
			t.Fatalf("expected verified certificate serial %q, got %q", "1234", service.serial)
		}
	})

	t.Run("rejects bad credentials", func(t *testing.T) {
		service := &deviceConnectionServiceStub{err: services.ErrDeviceAuthentication}
		router := gin.New()
		RegisterDeviceConnectionRoutes(router.Group("/api"), logger, service)

		request := httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/heartbeat", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
		}
	})

	t.Run("rejects expired certificates", func(t *testing.T) {
		service := &deviceConnectionServiceStub{err: services.ErrDeviceCertificateExpired}
		router := gin.New()
		RegisterDeviceConnectionRoutes(router.Group("/api"), logger, service)

		request := httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/disconnect", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
		}
	})

	t.Run("returns generic server error for storage failures", func(t *testing.T) {
		service := &deviceConnectionServiceStub{err: errors.New("database details")}
		router := gin.New()
		RegisterDeviceConnectionRoutes(router.Group("/api"), logger, service)

		request := httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/connect", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
		}
		if response.Body.String() == "" || response.Body.String() == "database details" {
			t.Fatalf("expected a generic error response, got %q", response.Body.String())
		}
	})
}
