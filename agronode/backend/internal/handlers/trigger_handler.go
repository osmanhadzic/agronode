package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"agronode/backend/internal/fuzzy"
	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type TriggerService interface {
	SetSensorTrigger(context context.Context, deviceID, sensorID string, trigger models.SensorTrigger) error
	GetSensorTrigger(context context.Context, deviceID, sensorID string) (models.SensorTrigger, error)
	ListSensorTriggers(context context.Context, deviceID string) (map[string]models.SensorTrigger, error)
	DeleteSensorTrigger(context context.Context, deviceID, sensorID string) error
}

type triggerHandler struct {
	logger  *slog.Logger
	service TriggerService
}

type sensorTriggerRequest struct {
	Min            *float64 `json:"min"`
	Max            *float64 `json:"max"`
	TargetDeviceID string   `json:"targetDeviceId"`
	FuzzyConfig    json.RawMessage `json:"fuzzyConfig,omitempty"`
}

type sensorTriggerResponse struct {
	DeviceID       string   `json:"deviceId"`
	Sensor         string   `json:"sensor,omitempty"`
	SensorID       string   `json:"sensorId"`
	Min            *float64 `json:"min,omitempty"`
	Max            *float64 `json:"max,omitempty"`
	TargetDeviceID string   `json:"targetDeviceId,omitempty"`
	FuzzyConfig    json.RawMessage `json:"fuzzyConfig,omitempty"`
}

type triggerListItem struct {
	Sensor         string   `json:"sensor,omitempty"`
	SensorID       string   `json:"sensorId"`
	Min            *float64 `json:"min,omitempty"`
	Max            *float64 `json:"max,omitempty"`
	TargetDeviceID string   `json:"targetDeviceId,omitempty"`
}

type triggerListResponse struct {
	DeviceID string            `json:"deviceId"`
	Triggers []triggerListItem `json:"triggers"`
}

func RegisterTriggerRoutes(api *gin.RouterGroup, logger *slog.Logger, service TriggerService) {
	handler := &triggerHandler{logger: logger, service: service}

	api.GET("/triggers/:deviceId", handler.listDeviceTriggers)
	api.PUT("/triggers/:deviceId/:sensorId", handler.setSensorTrigger)
	api.GET("/triggers/:deviceId/:sensorId", handler.getSensorTrigger)
	api.DELETE("/triggers/:deviceId/:sensorId", handler.deleteSensorTrigger)
	// Evaluate fuzzy trigger without performing activations - safe preview endpoint for UI
	api.POST("/triggers/:deviceId/:sensorId/evaluate", handler.evaluateFuzzyTrigger)
	api.POST("/triggers/:deviceId/:sensorId/evaluate-saved", handler.evaluateSavedTrigger)
}

type fuzzyEvaluateRequest struct {
	Inputs  map[string]float64   `json:"inputs"`
	Trigger fuzzy.FuzzyTrigger   `json:"trigger"`
}

// evaluateFuzzyTrigger accepts a fuzzy trigger and input values and returns evaluation details.
// This endpoint does NOT publish activation commands or change system state.
func (handler *triggerHandler) evaluateFuzzyTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensorID := context.Param("sensorId")
	// authorization / organization scope not required for preview, but keep consistent
	_, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req fuzzyEvaluateRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// fill missing membership function sensor fields with the requested sensorID for convenience
	for i := range req.Trigger.MembershipFunctions {
		if req.Trigger.MembershipFunctions[i].Sensor == "" {
			req.Trigger.MembershipFunctions[i].Sensor = sensorID
		}
	}

	// if inputs don't include the sensorID but contain a single unnamed value under "value", map it
	if req.Inputs == nil {
		req.Inputs = map[string]float64{}
	}

	result, err := fuzzy.EvaluateTrigger(req.Trigger, req.Inputs)
	if err != nil {
		if handler.logger != nil {
			handler.logger.Error("fuzzy evaluate failed", "deviceId", deviceID, "sensorId", sensorID, "error", err)
		}
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	context.JSON(http.StatusOK, result)
}

type evaluateSavedRequest struct {
	Inputs map[string]float64 `json:"inputs"`
}

func (handler *triggerHandler) evaluateSavedTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensorID := context.Param("sensorId")

	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req evaluateSavedRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	triggerModel, err := handler.service.GetSensorTrigger(requestContext, deviceID, sensorID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": "trigger not found"})
			return
		}
		handler.logger.Error("get sensor trigger failed", "deviceId", deviceID, "sensorId", sensorID, "error", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trigger"})
		return
	}

	if len(triggerModel.FuzzyConfig) == 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "trigger does not contain fuzzyConfig"})
		return
	}

	var ft fuzzy.FuzzyTrigger
	if err := json.Unmarshal(triggerModel.FuzzyConfig, &ft); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "invalid fuzzyConfig stored"})
		return
	}

	// fill missing membership function sensor fields with sensorID
	for i := range ft.MembershipFunctions {
		if ft.MembershipFunctions[i].Sensor == "" {
			ft.MembershipFunctions[i].Sensor = sensorID
		}
	}

	inputs := req.Inputs
	if inputs == nil {
		inputs = map[string]float64{}
	}

	result, err := fuzzy.EvaluateTrigger(ft, inputs)
	if err != nil {
		handler.logger.Error("fuzzy evaluate saved trigger failed", "deviceId", deviceID, "sensorId", sensorID, "error", err)
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	context.JSON(http.StatusOK, result)
}

func (handler *triggerHandler) deleteSensorTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensorID := context.Param("sensorId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = handler.service.DeleteSensorTrigger(requestContext, deviceID, sensorID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": "trigger not found"})
			return
		}

		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if handler.logger != nil {
			handler.logger.Error("delete sensor trigger failed", "deviceId", deviceID, "sensorId", sensorID, "error", err)
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete trigger"})
		return
	}

	context.Status(http.StatusNoContent)
}

func (handler *triggerHandler) listDeviceTriggers(context *gin.Context) {
	deviceID := context.Param("deviceId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	triggers, err := handler.service.ListSensorTriggers(requestContext, deviceID)
	if err != nil {
		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if handler.logger != nil {
			handler.logger.Error("list sensor triggers failed", "deviceId", deviceID, "error", err)
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch triggers"})
		return
	}

	items := make([]triggerListItem, 0, len(triggers))
	for sensorID, trigger := range triggers {
		items = append(items, triggerListItem{
			Sensor:         sensorID,
			SensorID:       sensorID,
			Min:            trigger.Min,
			Max:            trigger.Max,
			TargetDeviceID: trigger.TargetDeviceID,
		})
	}

	context.JSON(http.StatusOK, triggerListResponse{
		DeviceID: deviceID,
		Triggers: items,
	})
}

func (handler *triggerHandler) setSensorTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensorID := context.Param("sensorId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var request sensorTriggerRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	trigger := models.SensorTrigger{
		Min:            request.Min,
		Max:            request.Max,
		TargetDeviceID: request.TargetDeviceID,
		FuzzyConfig:    request.FuzzyConfig,
	}

	if err := handler.service.SetSensorTrigger(requestContext, deviceID, sensorID, trigger); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": "device not found in organization"})
			return
		}

		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if handler.logger != nil {
			handler.logger.Error("set sensor trigger failed", "deviceId", deviceID, "sensorId", sensorID, "error", err)
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set trigger"})
		return
	}

	if trigger.TargetDeviceID == "" {
		trigger.TargetDeviceID = deviceID
	}

	context.JSON(http.StatusOK, sensorTriggerResponse{
		DeviceID:       deviceID,
		Sensor:         sensorID,
		SensorID:       sensorID,
		Min:            trigger.Min,
		Max:            trigger.Max,
		TargetDeviceID: trigger.TargetDeviceID,
	})
}

func (handler *triggerHandler) getSensorTrigger(context *gin.Context) {
	deviceID := context.Param("deviceId")
	sensorID := context.Param("sensorId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trigger, err := handler.service.GetSensorTrigger(requestContext, deviceID, sensorID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			context.JSON(http.StatusOK, sensorTriggerResponse{
				DeviceID: deviceID,
				Sensor:   sensorID,
				SensorID: sensorID,
			})
			return
		}

		if errors.Is(err, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if handler.logger != nil {
			handler.logger.Error("get sensor trigger failed", "deviceId", deviceID, "sensorId", sensorID, "error", err)
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trigger"})
		return
	}

	context.JSON(http.StatusOK, sensorTriggerResponse{
		DeviceID:       deviceID,
		Sensor:         sensorID,
		SensorID:       sensorID,
		Min:            trigger.Min,
		Max:            trigger.Max,
		TargetDeviceID: trigger.TargetDeviceID,
	})
}
