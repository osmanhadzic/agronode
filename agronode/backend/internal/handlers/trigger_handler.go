package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type TriggerService interface {
	SetSensorTrigger(context context.Context, deviceID, sensor string, trigger models.SensorTrigger) error
	GetSensorTrigger(context context.Context, deviceID, sensor string) (models.SensorTrigger, error)
	ListSensorTriggers(context context.Context, deviceID string) (map[string]models.SensorTrigger, error)
	DeleteSensorTrigger(context context.Context, deviceID, sensor string) error
}

type triggerHandler struct {
	logger  *slog.Logger
	service TriggerService
}

type sensorTriggerRequest struct {
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
}

type sensorTriggerResponse struct {
	DeviceID string   `json:"deviceId"`
	Sensor   string   `json:"sensor"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

type triggerListItem struct {
	Sensor string   `json:"sensor"`
	Min    *float64 `json:"min,omitempty"`
	Max    *float64 `json:"max,omitempty"`
}

type triggerListResponse struct {
	DeviceID string            `json:"deviceId"`
	Triggers []triggerListItem `json:"triggers"`
}

func RegisterTriggerRoutes(api *gin.RouterGroup, logger *slog.Logger, service TriggerService) {
	handler := &triggerHandler{logger: logger, service: service}

	api.GET("/triggers/:deviceId", handler.listDeviceTriggers)
	api.PUT("/triggers/:deviceId/:sensor", handler.setSensorTrigger)
	api.GET("/triggers/:deviceId/:sensor", handler.getSensorTrigger)
	api.DELETE("/triggers/:deviceId/:sensor", handler.deleteSensorTrigger)
}

func (handler *triggerHandler) deleteSensorTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensor := context.Param("sensor")

	err := handler.service.DeleteSensorTrigger(context.Request.Context(), deviceID, sensor)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": "trigger not found"})
			return
		}

		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handler.logger.Error("delete sensor trigger failed", "deviceId", deviceID, "sensor", sensor, "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete trigger"})
		return
	}

	context.Status(http.StatusNoContent)
}

func (handler *triggerHandler) listDeviceTriggers(context *gin.Context) {
	deviceID := context.Param("deviceId")

	triggers, err := handler.service.ListSensorTriggers(context.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handler.logger.Error("list sensor triggers failed", "deviceId", deviceID, "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch triggers"})
		return
	}

	items := make([]triggerListItem, 0, len(triggers))
	for sensor, trigger := range triggers {
		items = append(items, triggerListItem{
			Sensor: sensor,
			Min:    trigger.Min,
			Max:    trigger.Max,
		})
	}

	context.JSON(http.StatusOK, triggerListResponse{
		DeviceID: deviceID,
		Triggers: items,
	})
}

func (handler *triggerHandler) setSensorTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensor := context.Param("sensor")

	var request sensorTriggerRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	trigger := models.SensorTrigger{
		Min: request.Min,
		Max: request.Max,
	}

	if err := handler.service.SetSensorTrigger(context.Request.Context(), deviceID, sensor, trigger); err != nil {
		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handler.logger.Error("set sensor trigger failed", "deviceId", deviceID, "sensor", sensor, "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set trigger"})
		return
	}

	context.JSON(http.StatusOK, sensorTriggerResponse{
		DeviceID: deviceID,
		Sensor:   sensor,
		Min:      request.Min,
		Max:      request.Max,
	})
}

func (handler *triggerHandler) getSensorTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensor := context.Param("sensor")

	trigger, err := handler.service.GetSensorTrigger(context.Request.Context(), deviceID, sensor)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			context.JSON(http.StatusOK, sensorTriggerResponse{
				DeviceID: deviceID,
				Sensor:   sensor,
			})
			return
		}

		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handler.logger.Error("get sensor trigger failed", "deviceId", deviceID, "sensor", sensor, "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trigger"})
		return
	}

	context.JSON(http.StatusOK, sensorTriggerResponse{
		DeviceID: deviceID,
		Sensor:   sensor,
		Min:      trigger.Min,
		Max:      trigger.Max,
	})
}
