package mqtt

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"agronode/backend/internal/models"
	"agronode/backend/internal/tenancy"
)

type testMessage struct {
	topic   string
	payload []byte
}

func (message testMessage) Duplicate() bool {
	return false
}

func (message testMessage) Qos() byte {
	return 1
}

func (message testMessage) Retained() bool {
	return false
}

func (message testMessage) Topic() string {
	return message.topic
}

func (message testMessage) MessageID() uint16 {
	return 1
}

func (message testMessage) Payload() []byte {
	return message.payload
}

func (message testMessage) Ack() {}

type telemetryConsumerStub struct {
	handled int
	last    TelemetryEnvelope
}

func (stub *telemetryConsumerStub) HandleTelemetry(_ context.Context, envelope TelemetryEnvelope) error {
	stub.handled++
	stub.last = envelope
	return nil
}

type deviceRegistrarStub struct {
	registerCalls int
	deviceID      string
	firmware      string
	metadata      models.DeviceMetadata
	tags          []string
	organization  uint
	err           error
}

func (stub *deviceRegistrarStub) RegisterDevice(ctx context.Context, deviceID string, firmwareVersion string, metadata models.DeviceMetadata, _ string, _ string, tags []string) (*models.Device, error) {
	if stub.err != nil {
		return nil, stub.err
	}

	stub.registerCalls++
	stub.deviceID = deviceID
	stub.firmware = firmwareVersion
	stub.metadata = metadata
	stub.tags = append([]string{}, tags...)
	if organizationID, ok := tenancy.OrganizationIDFromContext(ctx); ok {
		stub.organization = organizationID
	}
	return &models.Device{DeviceID: deviceID, FirmwareVersion: firmwareVersion}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestExtractDeviceIDFromTopic(t *testing.T) {
	t.Run("accepts telemetry topic", func(t *testing.T) {
		deviceID, err := extractDeviceIDFromTopic("agronode/Plastenik-1/telemetry")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if deviceID != "Plastenik-1" {
			t.Fatalf("expected device id %q, got %q", "Plastenik-1", deviceID)
		}
	})

	t.Run("accepts register topic", func(t *testing.T) {
		deviceID, err := extractDeviceIDFromTopic("agronode/Plastenik-1/register")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if deviceID != "Plastenik-1" {
			t.Fatalf("expected device id %q, got %q", "Plastenik-1", deviceID)
		}
	})

	t.Run("accepts status topic", func(t *testing.T) {
		deviceID, err := extractDeviceIDFromTopic("agronode/Plastenik-1/status")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if deviceID != "Plastenik-1" {
			t.Fatalf("expected device id %q, got %q", "Plastenik-1", deviceID)
		}
	})

	t.Run("rejects unsupported suffix", func(t *testing.T) {
		_, err := extractDeviceIDFromTopic("agronode/Plastenik-1/activation")
		if err == nil {
			t.Fatal("expected error for unsupported topic suffix")
		}
	})
}

func TestParseRegistrationPayload(t *testing.T) {
	t.Run("parses firmware and device id", func(t *testing.T) {
		payload, err := parseRegistrationPayload([]byte(`{"deviceId":"Plastenik-1","firmware":"1.0.1"}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if payload.DeviceID != "Plastenik-1" {
			t.Fatalf("expected device id %q, got %q", "Plastenik-1", payload.DeviceID)
		}

		if payload.FirmwareVersion != "1.0.1" {
			t.Fatalf("expected firmware %q, got %q", "1.0.1", payload.FirmwareVersion)
		}
	})

	t.Run("parses metadata and tags", func(t *testing.T) {
		payload, err := parseRegistrationPayload([]byte(`{"deviceId":"Plastenik-1","firmware":"1.0.1","metadata":{"signalStrength":-60,"hardware":{"model":"ESP32"}},"tags":["live","esp32"]}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if payload.Metadata.SignalStrength == nil || *payload.Metadata.SignalStrength != -60 {
			t.Fatalf("expected signalStrength -60, got %#v", payload.Metadata.SignalStrength)
		}

		if payload.Metadata.Hardware["model"] != "ESP32" {
			t.Fatalf("expected hardware model ESP32, got %q", payload.Metadata.Hardware["model"])
		}

		if len(payload.Tags) != 2 || payload.Tags[0] != "live" || payload.Tags[1] != "esp32" {
			t.Fatalf("expected tags [live esp32], got %v", payload.Tags)
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		_, err := parseRegistrationPayload([]byte(`{"deviceId":`))
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})
}

func TestParseTelemetryPayload(t *testing.T) {
	t.Run("parses sensor id and sensors map", func(t *testing.T) {
		payload, err := parseTelemetryPayload([]byte(`{"deviceId":"Plastenik-1","sensorId":"temperature","timestamp":1717243200,"sensors":{"temperature":24.5,"humidity":61}}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if payload.DeviceID != "Plastenik-1" {
			t.Fatalf("expected device id %q, got %q", "Plastenik-1", payload.DeviceID)
		}

		if payload.SensorID != "temperature" {
			t.Fatalf("expected sensor id %q, got %q", "temperature", payload.SensorID)
		}

		if payload.Sensors["temperature"] != 24.5 || payload.Sensors["humidity"] != 61 {
			t.Fatalf("unexpected sensors map: %#v", payload.Sensors)
		}
	})

	t.Run("treats legacy numeric root fields as sensors", func(t *testing.T) {
		payload, err := parseTelemetryPayload([]byte(`{"deviceId":"Plastenik-1","sensorId":"co2","co2":420}`))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if payload.SensorID != "co2" {
			t.Fatalf("expected sensor id %q, got %q", "co2", payload.SensorID)
		}

		if payload.Sensors["co2"] != 420 {
			t.Fatalf("expected co2 sensor value 420, got %#v", payload.Sensors)
		}
	})
}

func TestHandleMessage_ForwardsSensorID(t *testing.T) {
	consumer := &telemetryConsumerStub{}
	client := &Client{logger: testLogger(), consumer: consumer}

	message := testMessage{
		topic:   "agronode/Plastenik-1/telemetry",
		payload: []byte(`{"sensorId":"humidity","sensors":{"humidity":61}}`),
	}

	client.handleMessage(nil, message)

	if consumer.handled != 1 {
		t.Fatalf("expected telemetry handler to be called once, got %d", consumer.handled)
	}

	if consumer.last.SensorID != "humidity" {
		t.Fatalf("expected forwarded sensor id %q, got %q", "humidity", consumer.last.SensorID)
	}
}

func TestHandleMessage_RegisterTopic(t *testing.T) {
	t.Run("calls registrar and does not forward telemetry", func(t *testing.T) {
		consumer := &telemetryConsumerStub{}
		registrar := &deviceRegistrarStub{}
		client := &Client{logger: testLogger(), consumer: consumer, registrar: registrar}

		message := testMessage{
			topic:   "agronode/Plastenik-1/register",
			payload: []byte(`{"firmware":"1.0.1"}`),
		}

		client.handleMessage(nil, message)

		if registrar.registerCalls != 1 {
			t.Fatalf("expected 1 register call, got %d", registrar.registerCalls)
		}

		if registrar.deviceID != "Plastenik-1" {
			t.Fatalf("expected device id %q, got %q", "Plastenik-1", registrar.deviceID)
		}

		if registrar.firmware != "1.0.1" {
			t.Fatalf("expected firmware %q, got %q", "1.0.1", registrar.firmware)
		}

		if consumer.handled != 0 {
			t.Fatalf("expected telemetry handler not to be called, got %d", consumer.handled)
		}
	})

	t.Run("ignores register when registrar fails", func(t *testing.T) {
		consumer := &telemetryConsumerStub{}
		registrar := &deviceRegistrarStub{err: errors.New("db down")}
		client := &Client{logger: testLogger(), consumer: consumer, registrar: registrar}

		message := testMessage{
			topic:   "agronode/Plastenik-1/register",
			payload: []byte(`{"firmware":"1.0.1"}`),
		}

		client.handleMessage(nil, message)

		if consumer.handled != 0 {
			t.Fatalf("expected telemetry handler not to be called, got %d", consumer.handled)
		}
	})

	t.Run("passes default organization context and metadata", func(t *testing.T) {
		consumer := &telemetryConsumerStub{}
		registrar := &deviceRegistrarStub{}
		client := &Client{logger: testLogger(), consumer: consumer, registrar: registrar}
		client.SetDefaultOrganizationID(1)

		message := testMessage{
			topic:   "agronode/Plastenik-1/register",
			payload: []byte(`{"firmware":"1.0.1","metadata":{"signalStrength":-64},"tags":["live"]}`),
		}

		client.handleMessage(nil, message)

		if registrar.registerCalls != 1 {
			t.Fatalf("expected 1 register call, got %d", registrar.registerCalls)
		}

		if registrar.organization != 1 {
			t.Fatalf("expected organization id 1, got %d", registrar.organization)
		}

		if registrar.metadata.SignalStrength == nil || *registrar.metadata.SignalStrength != -64 {
			t.Fatalf("expected metadata signalStrength -64, got %#v", registrar.metadata.SignalStrength)
		}

		if len(registrar.tags) != 1 || registrar.tags[0] != "live" {
			t.Fatalf("expected tags [live], got %v", registrar.tags)
		}
	})
}
