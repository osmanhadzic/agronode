package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/repositories"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type TriggerExecutionService interface {
	ListTriggerExecutions(ctx context.Context, deviceID string, period string, startDate, endDate *time.Time) ([]models.IrrigationExecution, error)
}

type triggerEventResponse struct {
	RuleID       string `json:"ruleId"`
	DeviceID     string `json:"deviceId"`
	SourceDevice string `json:"sourceDeviceId,omitempty"`
	SensorID     string `json:"sensorId,omitempty"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
	TriggeredAt  string `json:"triggeredAt"`
	ErrorMessage string `json:"error,omitempty"`
}

func parseReasonField(reason, key string) string {
	for _, part := range strings.Fields(reason) {
		if value, found := strings.CutPrefix(part, key+"="); found {
			return value
		}
	}

	return ""
}

func (handler *telemetryHandler) getTriggerEvents(context *gin.Context) {
	deviceID := context.Param("deviceId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var startDate, endDate *time.Time
	if value := context.Query("startDate"); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid startDate format, use RFC3339"})
			return
		}
		startDate = &parsed
	}
	if value := context.Query("endDate"); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "invalid endDate format, use RFC3339"})
			return
		}
		endDate = &parsed
	}

	executions, serviceErr := handler.executions.ListTriggerExecutions(requestContext, deviceID, context.Query("period"), startDate, endDate)
	if serviceErr != nil {
		switch {
		case errors.Is(serviceErr, services.ErrValidation):
			context.JSON(http.StatusBadRequest, gin.H{"error": serviceErr.Error()})
		case errors.Is(serviceErr, repositories.ErrNotFound):
			context.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		default:
			if handler.logger != nil {
				handler.logger.Error("list trigger events failed", "deviceId", deviceID, "error", serviceErr)
			}
			context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trigger events"})
		}
		return
	}

	responses := make([]triggerEventResponse, 0, len(executions))
	for _, execution := range executions {
		response := triggerEventResponse{
			RuleID:       execution.RuleID,
			DeviceID:     execution.DeviceID,
			SourceDevice: parseReasonField(execution.Reason, "source_device"),
			SensorID:     parseReasonField(execution.Reason, "sensor"),
			Status:       execution.Status,
			Reason:       execution.Reason,
			TriggeredAt:  execution.TriggeredAt.UTC().Format(time.RFC3339),
		}
		if execution.ErrorMessage != nil {
			response.ErrorMessage = *execution.ErrorMessage
		}
		responses = append(responses, response)
	}

	context.JSON(http.StatusOK, responses)
}
