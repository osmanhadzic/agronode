package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppPort                     string
	LogLevel                    string
	LogFormat                   string
	LogService                  string
	LogEnvironment              string
	SessionSecret               string
	FrontendLoginEmail          string
	FrontendLoginPassword       string
	FrontendOrganizationID      uint
	FrontendSessionTTL          time.Duration
	DBHost                      string
	DBPort                      string
	DBUser                      string
	DBPassword                  string
	DBName                      string
	MQTTBroker                  string
	MQTTTopic                   string
	DeviceInactivityMin         string
	DeviceWorkerInterval        string
	MQTTActivationTopicTemplate string
	AdminChatLLMEnabled         bool
	AdminChatLLMProvider        string
	AdminChatLLMBaseURL         string
	AdminChatLLMAPIKey          string
	AdminChatLLMModel           string
	AdminChatLLMChatPath        string
	AdminChatLLMSystemPrompt    string
	AdminChatLLMTimeoutSeconds  int
}

func Load() Config {
	return Config{
		AppPort:                     getEnv("APP_PORT", "8080"),
		LogLevel:                    getEnv("LOG_LEVEL", "info"),
		LogFormat:                   getEnv("LOG_FORMAT", "json"),
		LogService:                  getEnv("LOG_SERVICE", "agronode-backend"),
		LogEnvironment:              getEnv("LOG_ENV", "dev"),
		SessionSecret:               getEnv("SESSION_SECRET", "dev-session-secret"),
		FrontendLoginEmail:          getEnv("FRONTEND_LOGIN_EMAIL", "admin@agronode.local"),
		FrontendLoginPassword:       getEnv("FRONTEND_LOGIN_PASSWORD", "admin123"),
		FrontendOrganizationID:      getEnvUint("FRONTEND_ORGANIZATION_ID", 1),
		FrontendSessionTTL:          getEnvDurationHours("FRONTEND_SESSION_TTL_HOURS", 24),
		DBHost:                      getEnv("DB_HOST", "postgres"),
		DBPort:                      getEnv("DB_PORT", "5432"),
		DBUser:                      getEnv("DB_USER", "postgres"),
		DBPassword:                  getEnv("DB_PASSWORD", "postgres"),
		DBName:                      getEnv("DB_NAME", "agronode"),
		MQTTBroker:                  getEnv("MQTT_BROKER", "tcp://mosquitto:1883"),
		MQTTTopic:                   getEnv("MQTT_TOPIC", "agronode/#"),
		DeviceInactivityMin:         getEnv("DEVICE_INACTIVITY_MIN", "15"),
		DeviceWorkerInterval:        getEnv("DEVICE_WORKER_INTERVAL", "1"),
		MQTTActivationTopicTemplate: getEnv("MQTT_ACTIVATION_TOPIC_TEMPLATE", "agronode/%s/activation"),
		AdminChatLLMEnabled:         getEnvBool("ADMIN_CHAT_LLM_ENABLED", false),
		AdminChatLLMProvider:        getEnv("ADMIN_CHAT_LLM_PROVIDER", "ollama"),
		AdminChatLLMBaseURL:         getEnv("ADMIN_CHAT_LLM_BASE_URL", "http://ollama:11434"),
		AdminChatLLMAPIKey:          getEnv("ADMIN_CHAT_LLM_API_KEY", ""),
		AdminChatLLMModel:           getEnv("ADMIN_CHAT_LLM_MODEL", "llama3.2:1b"),
		AdminChatLLMChatPath:        getEnv("ADMIN_CHAT_LLM_CHAT_COMPLETIONS_PATH", "/v1/chat/completions"),
		AdminChatLLMSystemPrompt:    getEnv("ADMIN_CHAT_LLM_SYSTEM_PROMPT", "You are AgroNode Admin Assistant. Keep answers short, practical, and focused on device telemetry, triggers, MQTT, and operations."),
		AdminChatLLMTimeoutSeconds:  getEnvInt("ADMIN_CHAT_LLM_TIMEOUT_SECONDS", 20),
	}
}

func NewLogger(level, format, service, environment string) *slog.Logger {
	logLevel := slog.LevelInfo

	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(format), "text") {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	}

	logger := slog.New(handler)
	if strings.TrimSpace(service) != "" {
		logger = logger.With("service", strings.TrimSpace(service))
	}
	if strings.TrimSpace(environment) != "" {
		logger = logger.With("env", strings.TrimSpace(environment))
	}

	return logger
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvUint(key string, fallback uint) uint {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return fallback
	}

	return uint(parsed)
}

func getEnvDurationHours(key string, fallbackHours int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(fallbackHours) * time.Hour
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return time.Duration(fallbackHours) * time.Hour
	}

	return time.Duration(parsed) * time.Hour
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
