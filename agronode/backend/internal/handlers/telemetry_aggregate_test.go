package handlers

import (
	"testing"
	"time"

	"agronode/backend/internal/models"
)

func TestAggregateTelemetryBucketsAndSummary(t *testing.T) {
	base := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	readings := []models.TelemetryReading{
		{DeviceID: "d", Sensors: map[string]float64{"temp": 10}, CreatedAt: base.Add(1 * time.Minute)},
		{DeviceID: "d", Sensors: map[string]float64{"temp": 20}, CreatedAt: base.Add(30 * time.Minute)},
		{DeviceID: "d", Sensors: map[string]float64{"temp": 30}, CreatedAt: base.Add(61 * time.Minute)},
	}

	buckets, summary := aggregateTelemetry(readings, time.Hour)

	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
	first := buckets[0].Sensors["temp"]
	if first.Min != 10 || first.Max != 20 || first.Avg != 15 || first.Count != 2 {
		t.Fatalf("unexpected first bucket stats: %+v", first)
	}
	all := summary["temp"]
	if all.Min != 10 || all.Max != 30 || all.Avg != 20 || all.Count != 3 {
		t.Fatalf("unexpected summary: %+v", all)
	}
}

func TestResolveBucketDuration(t *testing.T) {
	if name, d := resolveBucketDuration("", "day"); name != "15m" || d != 15*time.Minute {
		t.Fatalf("unexpected default for day: %s %v", name, d)
	}
	if name, _ := resolveBucketDuration("hour", "year"); name != "hour" {
		t.Fatalf("explicit bucket should win, got %s", name)
	}
}
