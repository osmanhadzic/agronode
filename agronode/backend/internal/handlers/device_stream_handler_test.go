package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type deviceStreamServiceStub struct {
	err error
}

func (stub *deviceStreamServiceStub) SetTelemetryStreaming(_ context.Context, _ string, _ string, _ bool) error {
	return stub.err
}

func TestDeviceStreamHandler_setStreamControl(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns bad request on invalid action", func(t *testing.T) {
		service := &deviceStreamServiceStub{}
		handler := &deviceStreamHandler{service: service}

		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/stream-control", bytes.NewBufferString(`{"action":"invalid"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = gin.Params{{Key: "deviceId", Value: "esp32-lab"}}
		ctx.Request = ctx.Request.WithContext(context.Background())
		ctx.Request.Header.Set(organizationIDHeader, "1")

		handler.setStreamControl(ctx)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
		}
	})

	t.Run("returns bad request on validation error", func(t *testing.T) {
		service := &deviceStreamServiceStub{err: fmt.Errorf("%w: device id is required", services.ErrValidation)}
		handler := &deviceStreamHandler{service: service}

		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/stream-control", bytes.NewBufferString(`{"action":"pause"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = gin.Params{{Key: "deviceId", Value: "esp32-lab"}}
		ctx.Request = ctx.Request.WithContext(context.Background())
		ctx.Request.Header.Set(organizationIDHeader, "1")

		handler.setStreamControl(ctx)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
		}
	})

	t.Run("returns ok on pause action", func(t *testing.T) {
		service := &deviceStreamServiceStub{}
		handler := &deviceStreamHandler{service: service}

		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/devices/esp32-lab/stream-control", bytes.NewBufferString(`{"action":"pause"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = gin.Params{{Key: "deviceId", Value: "esp32-lab"}}
		ctx.Request = ctx.Request.WithContext(context.Background())
		ctx.Request.Header.Set(organizationIDHeader, "1")

		handler.setStreamControl(ctx)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}
	})
}
