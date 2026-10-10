package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agronode/backend/internal/config"
)

type AdminChatLLMService interface {
	Enabled() bool
	GenerateReply(ctx context.Context, userMessage string) (string, error)
}

type NoopAdminChatLLMService struct{}

func NewNoopAdminChatLLMService() *NoopAdminChatLLMService {
	return &NoopAdminChatLLMService{}
}

func (service *NoopAdminChatLLMService) Enabled() bool {
	return false
}

func (service *NoopAdminChatLLMService) GenerateReply(context.Context, string) (string, error) {
	return "", nil
}

type ollamaAdminChatLLMService struct {
	httpClient   *http.Client
	baseURL      string
	model        string
	systemPrompt string
}

type openAIAdminChatLLMService struct {
	httpClient   *http.Client
	baseURL      string
	apiKey       string
	chatPath     string
	model        string
	systemPrompt string
}

func NewAdminChatLLMService(cfg config.Config) AdminChatLLMService {
	if !cfg.AdminChatLLMEnabled {
		return NewNoopAdminChatLLMService()
	}

	provider := strings.TrimSpace(strings.ToLower(cfg.AdminChatLLMProvider))
	if provider == "" {
		provider = "ollama"
	}

	timeoutSeconds := cfg.AdminChatLLMTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 20
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.AdminChatLLMBaseURL), "/")
	if baseURL == "" {
		baseURL = "http://ollama:11434"
	}

	model := strings.TrimSpace(cfg.AdminChatLLMModel)
	if model == "" {
		model = "llama3.2:1b"
	}

	systemPrompt := strings.TrimSpace(cfg.AdminChatLLMSystemPrompt)
	if systemPrompt == "" {
		systemPrompt = "You are AgroNode Admin Assistant. Keep answers short, practical, and focused on device telemetry, triggers, MQTT, and operations."
	}

	if provider == "openai" || provider == "openai-compatible" || provider == "external" {
		chatPath := strings.TrimSpace(cfg.AdminChatLLMChatPath)
		if chatPath == "" {
			chatPath = "/v1/chat/completions"
		}

		if !strings.HasPrefix(chatPath, "/") {
			chatPath = "/" + chatPath
		}

		return &openAIAdminChatLLMService{
			httpClient: &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
			baseURL: baseURL,
			apiKey: strings.TrimSpace(cfg.AdminChatLLMAPIKey),
			chatPath: chatPath,
			model: model,
			systemPrompt: systemPrompt,
		}
	}

	if provider != "ollama" {
		return NewNoopAdminChatLLMService()
	}

	return &ollamaAdminChatLLMService{
		httpClient: &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		baseURL:    baseURL,
		model:      model,
		systemPrompt: systemPrompt,
	}
}

func (service *ollamaAdminChatLLMService) Enabled() bool {
	return true
}

type ollamaChatRequest struct {
	Model    string               `json:"model"`
	Messages []ollamaChatMessage  `json:"messages"`
	Stream   bool                 `json:"stream"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaChatMessage `json:"message"`
}

func (service *ollamaAdminChatLLMService) GenerateReply(ctx context.Context, userMessage string) (string, error) {
	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return "", nil
	}

	requestBody := ollamaChatRequest{
		Model: service.model,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: service.systemPrompt},
			{Role: "user", Content: trimmed},
		},
		Stream: false,
	}

	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, service.baseURL+"/api/chat", bytes.NewReader(encodedBody))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := service.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("ollama request failed with status %d", response.StatusCode)
	}

	var parsed ollamaChatResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&parsed); decodeErr != nil {
		return "", decodeErr
	}

	content := strings.TrimSpace(parsed.Message.Content)
	if content == "" {
		return "", nil
	}

	if len(content) > 1000 {
		content = content[:1000]
	}

	return content, nil
}

type openAIChatRequest struct {
	Model    string               `json:"model"`
	Messages []ollamaChatMessage  `json:"messages"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message ollamaChatMessage `json:"message"`
	} `json:"choices"`
}

func (service *openAIAdminChatLLMService) Enabled() bool {
	return true
}

func (service *openAIAdminChatLLMService) GenerateReply(ctx context.Context, userMessage string) (string, error) {
	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return "", nil
	}

	requestBody := openAIChatRequest{
		Model: service.model,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: service.systemPrompt},
			{Role: "user", Content: trimmed},
		},
	}

	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(service.baseURL, "/")+service.chatPath, bytes.NewReader(encodedBody))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if service.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+service.apiKey)
	}

	response, err := service.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var errorPayload map[string]any
		if decodeErr := json.NewDecoder(response.Body).Decode(&errorPayload); decodeErr == nil {
			if errorObject, ok := errorPayload["error"].(map[string]any); ok {
				if message, ok := errorObject["message"].(string); ok && strings.TrimSpace(message) != "" {
					return "", fmt.Errorf("openai-compatible request failed with status %d: %s", response.StatusCode, strings.TrimSpace(message))
				}
			}
		}

		return "", fmt.Errorf("openai-compatible request failed with status %d", response.StatusCode)
	}

	var parsed openAIChatResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&parsed); decodeErr != nil {
		return "", decodeErr
	}

	if len(parsed.Choices) == 0 {
		return "", nil
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", nil
	}

	if len(content) > 1000 {
		content = content[:1000]
	}

	return content, nil
}
