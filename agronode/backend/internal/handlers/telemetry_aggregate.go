package handlers

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"agronode/backend/internal/models"
	"agronode/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type sensorStatsResponse struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Count int     `json:"count"`
}

type telemetryBucketResponse struct {
	Time    string                         `json:"time"`
	Sensors map[string]sensorStatsResponse `json:"sensors"`
}

type telemetryAggregateResponse struct {
	DeviceID string                         `json:"deviceId"`
	Bucket   string                         `json:"bucket"`
	Buckets  []telemetryBucketResponse      `json:"buckets"`
	Summary  map[string]sensorStatsResponse `json:"summary"`
}

type statsAccumulator struct {
	min, max, sum float64
	count         int
}

func (acc *statsAccumulator) add(value float64) {
	if acc.count == 0 || value < acc.min {
		acc.min = value
	}
	if acc.count == 0 || value > acc.max {
		acc.max = value
	}
	acc.sum += value
	acc.count++
}

func (acc *statsAccumulator) response() sensorStatsResponse {
	return sensorStatsResponse{Min: acc.min, Max: acc.max, Avg: acc.sum / float64(acc.count), Count: acc.count}
}

func resolveBucketDuration(bucket, period string) (string, time.Duration) {
	switch bucket {
	case "minute":
		return bucket, time.Minute
	case "5m":
		return bucket, 5 * time.Minute
	case "15m":
		return bucket, 15 * time.Minute
	case "hour":
		return bucket, time.Hour
	case "day":
		return bucket, 24 * time.Hour
	}

	switch period {
	case "hour":
		return "minute", time.Minute
	case "day":
		return "15m", 15 * time.Minute
	case "week":
		return "hour", time.Hour
	case "month", "year":
		return "day", 24 * time.Hour
	default:
		return "hour", time.Hour
	}
}

func readingSensorValues(reading models.TelemetryReading) map[string]float64 {
	if len(reading.Sensors) > 0 {
		return reading.Sensors
	}

	values := map[string]float64{}
	if reading.SensorID == "" {
		values["temperature"] = reading.Temperature
		values["humidity"] = reading.Humidity
		return values
	}

	if value, ok := telemetryValueForKeys(reading, reading.SensorID); ok {
		values[reading.SensorID] = value
	}

	return values
}

func aggregateTelemetry(readings []models.TelemetryReading, bucketSize time.Duration) ([]telemetryBucketResponse, map[string]sensorStatsResponse) {
	buckets := map[int64]map[string]*statsAccumulator{}
	summary := map[string]*statsAccumulator{}

	for _, reading := range readings {
		start := reading.CreatedAt.UTC().Truncate(bucketSize).Unix()
		if buckets[start] == nil {
			buckets[start] = map[string]*statsAccumulator{}
		}

		for sensorID, value := range readingSensorValues(reading) {
			if buckets[start][sensorID] == nil {
				buckets[start][sensorID] = &statsAccumulator{}
			}
			if summary[sensorID] == nil {
				summary[sensorID] = &statsAccumulator{}
			}
			buckets[start][sensorID].add(value)
			summary[sensorID].add(value)
		}
	}

	starts := make([]int64, 0, len(buckets))
	for start := range buckets {
		starts = append(starts, start)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })

	result := make([]telemetryBucketResponse, 0, len(starts))
	for _, start := range starts {
		sensors := make(map[string]sensorStatsResponse, len(buckets[start]))
		for sensorID, acc := range buckets[start] {
			sensors[sensorID] = acc.response()
		}
		result = append(result, telemetryBucketResponse{
			Time:    time.Unix(start, 0).UTC().Format(time.RFC3339),
			Sensors: sensors,
		})
	}

	summaryResponse := make(map[string]sensorStatsResponse, len(summary))
	for sensorID, acc := range summary {
		summaryResponse[sensorID] = acc.response()
	}

	return result, summaryResponse
}

func (handler *telemetryHandler) getAggregatedData(context *gin.Context) {
	deviceID := context.Param("deviceId")
	requestContext, err := requestContextWithOrganizationScope(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	period := context.Query("period")
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

	bucketName, bucketSize := resolveBucketDuration(context.Query("bucket"), period)

	readings, serviceErr := handler.service.GetTelemetryByDeviceIDWithDateFilter(requestContext, deviceID, period, startDate, endDate)
	if serviceErr != nil {
		if handler.logger != nil {
			handler.logger.Error("aggregate telemetry failed", "deviceId", deviceID, "error", serviceErr)
		}
		if errors.Is(serviceErr, services.ErrValidation) {
			context.JSON(http.StatusBadRequest, gin.H{"error": serviceErr.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch telemetry"})
		return
	}

	buckets, summary := aggregateTelemetry(readings, bucketSize)
	context.JSON(http.StatusOK, telemetryAggregateResponse{
		DeviceID: deviceID,
		Bucket:   bucketName,
		Buckets:  buckets,
		Summary:  summary,
	})
}
