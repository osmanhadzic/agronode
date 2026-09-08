package mqtt

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"agronode/backend/internal/models"
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
}

func (stub *telemetryConsumerStub) HandleTelemetry(_ context.Context, _ TelemetryEnvelope) error {
	stub.handled++
	return nil
}

type deviceRegistrarStub struct {
	registerCalls int
	deviceID      string
	firmware      string
	err           error
}

func (stub *deviceRegistrarStub) RegisterDevice(_ context.Context, deviceID string, firmwareVersion string, _ models.DeviceMetadata, _ string, _ string, _ []string) (*models.Device, error) {
	if stub.err != nil {
		return nil, stub.err
	}

	stub.registerCalls++
	stub.deviceID = deviceID
	stub.firmware = firmwareVersion
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

	t.Run("rejects invalid json", func(t *testing.T) {
		_, err := parseRegistrationPayload([]byte(`{"deviceId":`))
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})
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
}
