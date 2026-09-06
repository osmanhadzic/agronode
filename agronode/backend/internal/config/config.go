package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	AppPort                     string
	LogLevel                    string
	DBHost                      string
	DBPort                      string
	DBUser                      string
	DBPassword                  string
	DBName                      string
	MQTTBroker                  string
	MQTTTopic                   string
	MQTTActivationTopicTemplate string
}

func Load() Config {
	return Config{
		AppPort:                     getEnv("APP_PORT", "8080"),
		LogLevel:                    getEnv("LOG_LEVEL", "info"),
		DBHost:                      getEnv("DB_HOST", "postgres"),
		DBPort:                      getEnv("DB_PORT", "5432"),
		DBUser:                      getEnv("DB_USER", "postgres"),
		DBPassword:                  getEnv("DB_PASSWORD", "postgres"),
		DBName:                      getEnv("DB_NAME", "agronode"),
		MQTTBroker:                  getEnv("MQTT_BROKER", "tcp://mosquitto:1883"),
		MQTTTopic:                   getEnv("MQTT_TOPIC", "agronode/#"),
		MQTTActivationTopicTemplate: getEnv("MQTT_ACTIVATION_TOPIC_TEMPLATE", "agronode/%s/activation"),
	}
}

func NewLogger(level string) *slog.Logger {
	logLevel := slog.LevelInfo

	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
