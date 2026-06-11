package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type DeviceMeta struct {
	Firmware string `json:"fw"`
	IP       string `json:"ip"`
	RSSI     int    `json:"rssi"`
	Uptime   uint64 `json:"uptime"`
}

type TelemetryEnvelope struct {
	Topic           string
	DeviceID        string
	PayloadDeviceID string
	Timestamp       int64
	Sensors         map[string]float64
	Meta            *DeviceMeta
}

type TelemetryConsumer interface {
	HandleTelemetry(context.Context, TelemetryEnvelope) error
}

type telemetryPayload struct {
	DeviceID  string             `json:"deviceId"`
	Timestamp int64              `json:"timestamp"`
	Sensors   map[string]float64 `json:"sensors"`
	Meta      *DeviceMeta        `json:"meta"`
}

type Client struct {
	brokerURL string
	topic     string
	logger    *slog.Logger
	consumer  TelemetryConsumer
	client    paho.Client
}

func NewClient(brokerURL, topic string, logger *slog.Logger, consumer TelemetryConsumer) *Client {
	return &Client{
		brokerURL: brokerURL,
		topic:     topic,
		logger:    logger,
		consumer:  consumer,
	}
}

func (client *Client) Run(runContext context.Context) error {
	options := paho.NewClientOptions()
	options.AddBroker(client.brokerURL)
	options.SetClientID(fmt.Sprintf("agronode-backend-%d", time.Now().UnixNano()))
	options.SetAutoReconnect(true)
	options.SetConnectRetry(true)
	options.SetConnectRetryInterval(2 * time.Second)

	options.OnConnect = func(connectedClient paho.Client) {
		client.logger.Info("mqtt connected", "broker", client.brokerURL)

		token := connectedClient.Subscribe(client.topic, 1, client.handleMessage)
		token.Wait()
		if token.Error() != nil {
			client.logger.Error("mqtt subscribe failed", "topic", client.topic, "error", token.Error())
			return
		}

		client.logger.Info("mqtt subscribed", "topic", client.topic)
	}

	options.OnConnectionLost = func(_ paho.Client, err error) {
		client.logger.Warn("mqtt connection lost", "error", err)
	}

	client.client = paho.NewClient(options)

	connectToken := client.client.Connect()
	connectToken.Wait()
	if connectToken.Error() != nil {
		return fmt.Errorf("mqtt connect: %w", connectToken.Error())
	}

	<-runContext.Done()

	if client.client.IsConnected() {
		client.client.Disconnect(250)
	}

	client.logger.Info("mqtt client stopped")
	return nil
}

func (client *Client) handleMessage(_ paho.Client, message paho.Message) {
	deviceID, err := extractDeviceIDFromTopic(message.Topic())
	if err != nil {
		client.logger.Warn("mqtt topic rejected", "topic", message.Topic(), "error", err)
		return
	}

	payload, err := parseTelemetryPayload(message.Payload())
	if err != nil {
		client.logger.Warn("mqtt payload rejected", "topic", message.Topic(), "error", err)
		return
	}

	if payload.DeviceID != "" && payload.DeviceID != deviceID {
		client.logger.Warn("mqtt payload device mismatch", "topicDeviceID", deviceID, "payloadDeviceID", payload.DeviceID)
	}

	envelope := TelemetryEnvelope{
		Topic:           message.Topic(),
		DeviceID:        deviceID,
		PayloadDeviceID: payload.DeviceID,
		Timestamp:       payload.Timestamp,
		Sensors:         payload.Sensors,
		Meta:            payload.Meta,
	}

	if client.consumer == nil {
		client.logger.Warn("mqtt message dropped: no service consumer configured", "topic", message.Topic())
		return
	}

	if err := client.consumer.HandleTelemetry(context.Background(), envelope); err != nil {
		client.logger.Error("service layer forwarding failed", "topic", message.Topic(), "error", err)
		return
	}
}

func extractDeviceIDFromTopic(topic string) (string, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("topic must contain exactly 3 segments")
	}

	if parts[0] != "agronode" || parts[2] != "telemetry" {
		return "", fmt.Errorf("topic must match agronode/{deviceId}/telemetry")
	}

	deviceID := strings.TrimSpace(parts[1])
	if deviceID == "" {
		return "", fmt.Errorf("device id segment is empty")
	}

	return deviceID, nil
}

func parseTelemetryPayload(payloadBytes []byte) (telemetryPayload, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		return telemetryPayload{}, err
	}

	payload := telemetryPayload{Sensors: map[string]float64{}}

	if deviceIDBytes, hasDeviceID := raw["deviceId"]; hasDeviceID {
		if err := json.Unmarshal(deviceIDBytes, &payload.DeviceID); err != nil {
			return telemetryPayload{}, fmt.Errorf("invalid deviceId: %w", err)
		}
	}

	if timestampBytes, hasTimestamp := raw["timestamp"]; hasTimestamp {
		timestamp, err := parseTimestamp(timestampBytes)
		if err != nil {
			return telemetryPayload{}, err
		}

		payload.Timestamp = timestamp
	}

	if sensorsBytes, hasSensors := raw["sensors"]; hasSensors {
		sensors, err := parseSensorsMap(sensorsBytes)
		if err != nil {
			return telemetryPayload{}, err
		}

		for key, value := range sensors {
			payload.Sensors[key] = value
		}
	}

	if metaBytes, hasMeta := raw["meta"]; hasMeta {
		var meta DeviceMeta
		if err := json.Unmarshal(metaBytes, &meta); err == nil {
			payload.Meta = &meta
		}
	}

	if len(payload.Sensors) == 0 {
		for key, valueBytes := range raw {
			if key == "deviceId" || key == "timestamp" || key == "sensors" || key == "version" {
				continue
			}

			value, ok, err := parseNumericRaw(valueBytes)
			if err != nil {
				return telemetryPayload{}, fmt.Errorf("invalid numeric field %s: %w", key, err)
			}

			if ok {
				payload.Sensors[key] = value
			}
		}
	}

	return payload, nil
}

func parseTimestamp(rawTimestamp json.RawMessage) (int64, error) {
	var asInt int64
	if err := json.Unmarshal(rawTimestamp, &asInt); err == nil {
		return asInt, nil
	}

	var asFloat float64
	if err := json.Unmarshal(rawTimestamp, &asFloat); err == nil {
		return int64(asFloat), nil
	}

	var asString string
	if err := json.Unmarshal(rawTimestamp, &asString); err == nil {
		parsedValue, parseErr := strconv.ParseInt(strings.TrimSpace(asString), 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("invalid timestamp string")
		}

		return parsedValue, nil
	}

	return 0, fmt.Errorf("invalid timestamp format")
}

func parseSensorsMap(rawSensors json.RawMessage) (map[string]float64, error) {
	var sensorsRaw map[string]json.RawMessage
	if err := json.Unmarshal(rawSensors, &sensorsRaw); err != nil {
		return nil, fmt.Errorf("invalid sensors object: %w", err)
	}

	sensors := make(map[string]float64, len(sensorsRaw))
	for key, valueBytes := range sensorsRaw {
		value, ok, err := parseNumericRaw(valueBytes)
		if err != nil {
			return nil, fmt.Errorf("invalid sensor %s: %w", key, err)
		}

		if ok {
			sensors[key] = value
		}
	}

	return sensors, nil
}

func parseNumericRaw(rawValue json.RawMessage) (float64, bool, error) {
	var asFloat float64
	if err := json.Unmarshal(rawValue, &asFloat); err == nil {
		return asFloat, true, nil
	}

	var asString string
	if err := json.Unmarshal(rawValue, &asString); err == nil {
		parsedValue, parseErr := strconv.ParseFloat(strings.TrimSpace(asString), 64)
		if parseErr != nil {
			return 0, false, fmt.Errorf("not a number")
		}

		return parsedValue, true, nil
	}

	return 0, false, nil
}
