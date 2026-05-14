package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type TelemetryEnvelope struct {
	Topic           string
	DeviceID        string
	PayloadDeviceID string
	Timestamp       int64
	Sensors         map[string]float64
}

type TelemetryConsumer interface {
	HandleTelemetry(context.Context, TelemetryEnvelope) error
}

type telemetryPayload struct {
	DeviceID  string             `json:"deviceId"`
	Timestamp int64              `json:"timestamp"`
	Sensors   map[string]float64 `json:"sensors"`
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
	var payload telemetryPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return telemetryPayload{}, err
	}

	if payload.Sensors == nil {
		payload.Sensors = map[string]float64{}
	}

	return payload, nil
}
