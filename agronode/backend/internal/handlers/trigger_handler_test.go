package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type triggerServiceStub struct {
	setErr    error
	getErr    error
	listErr   error
	deleteErr error
}

func (stub *triggerServiceStub) SetSensorTrigger(_ context.Context, _ string, _ string, _ models.SensorTrigger) error {
	return stub.setErr
}

func (stub *triggerServiceStub) GetSensorTrigger(_ context.Context, _, _ string) (models.SensorTrigger, error) {
	return models.SensorTrigger{}, stub.getErr
}

func (stub *triggerServiceStub) ListSensorTriggers(_ context.Context, _ string) (map[string]models.SensorTrigger, error) {
	return nil, stub.listErr
}

func (stub *triggerServiceStub) DeleteSensorTrigger(_ context.Context, _, _ string) error {
	return stub.deleteErr
}

func TestTriggerHandler_setSensorTrigger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns not found when device is outside organization scope", func(t *testing.T) {
		service := &triggerServiceStub{setErr: repositories.ErrNotFound}
		handler := &triggerHandler{service: service}

		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/triggers/esp32-lab/co2", bytes.NewBufferString(`{"max":700}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = gin.Params{{Key: "deviceId", Value: "esp32-lab"}, {Key: "sensorId", Value: "co2"}}
		ctx.Request = ctx.Request.WithContext(context.Background())
		ctx.Request.Header.Set(organizationIDHeader, "1")

		handler.setSensorTrigger(ctx)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
		}
	})

	t.Run("returns bad request on validation error", func(t *testing.T) {
		service := &triggerServiceStub{setErr: fmt.Errorf("%w: device id is required", services.ErrValidation)}
		handler := &triggerHandler{service: service}

		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/triggers/esp32-lab/co2", bytes.NewBufferString(`{"max":700}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = gin.Params{{Key: "deviceId", Value: "esp32-lab"}, {Key: "sensorId", Value: "co2"}}
		ctx.Request = ctx.Request.WithContext(context.Background())
		ctx.Request.Header.Set(organizationIDHeader, "1")

		handler.setSensorTrigger(ctx)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
		}
	})
}
