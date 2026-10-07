using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using AgroNode.McpServer.Options;

namespace AgroNode.McpServer.Clients;

internal sealed class NoopLlmClient : ILlmClient
{
    public bool IsEnabled => false;

    public Task<string> AskAsync(string prompt, string? systemPromptOverride, CancellationToken cancellationToken)
    {
        _ = prompt;
        _ = systemPromptOverride;
        _ = cancellationToken;
        throw new InvalidOperationException("LLM is disabled for this MCP server.");
    }
}

internal sealed class OllamaLlmClient(HttpClient httpClient, LlmOptions options) : ILlmClient
{
    public bool IsEnabled => options.Enabled;

    public async Task<string> AskAsync(string prompt, string? systemPromptOverride, CancellationToken cancellationToken)
    {
        var trimmedPrompt = prompt.Trim();
        if (string.IsNullOrWhiteSpace(trimmedPrompt))
        {
            throw new InvalidOperationException("Argument 'prompt' is required.");
        }

        var systemPrompt = string.IsNullOrWhiteSpace(systemPromptOverride)
            ? options.SystemPrompt
            : systemPromptOverride.Trim();

        var payload = new
        {
            model = options.Model,
            stream = false,
            messages = new object[]
            {
                new { role = "system", content = systemPrompt },
                new { role = "user", content = trimmedPrompt }
            }
        };

        using var request = new HttpRequestMessage(HttpMethod.Post, "/api/chat")
        {
            Content = new StringContent(JsonSerializer.Serialize(payload), Encoding.UTF8, "application/json")
        };

        using var response = await httpClient.SendAsync(request, cancellationToken);
        var body = await response.Content.ReadAsStringAsync(cancellationToken);

        if (!response.IsSuccessStatusCode)
        {
            throw new InvalidOperationException($"LLM request failed ({(int)response.StatusCode}): {body}");
        }

        using var document = JsonDocument.Parse(body);
        var content = document.RootElement
            .GetProperty("message")
            .GetProperty("content")
            .GetString();

        if (string.IsNullOrWhiteSpace(content))
        {
            throw new InvalidOperationException("LLM returned an empty response.");
        }

        return content.Trim();
    }
}

internal sealed class OpenAiCompatibleLlmClient(HttpClient httpClient, LlmOptions options) : ILlmClient
{
    public bool IsEnabled => options.Enabled;

    public async Task<string> AskAsync(string prompt, string? systemPromptOverride, CancellationToken cancellationToken)
    {
        var trimmedPrompt = prompt.Trim();
        if (string.IsNullOrWhiteSpace(trimmedPrompt))
        {
            throw new InvalidOperationException("Argument 'prompt' is required.");
        }

        var systemPrompt = string.IsNullOrWhiteSpace(systemPromptOverride)
            ? options.SystemPrompt
            : systemPromptOverride.Trim();

        var requestBody = new JsonObject
        {
            ["model"] = options.Model,
            ["messages"] = new JsonArray
            {
                new JsonObject
                {
                    ["role"] = "system",
                    ["content"] = systemPrompt
                },
                new JsonObject
                {
                    ["role"] = "user",
                    ["content"] = trimmedPrompt
                }
            }
        };

        using var request = new HttpRequestMessage(HttpMethod.Post, options.ChatCompletionsPath)
        {
            Content = new StringContent(requestBody.ToJsonString(), Encoding.UTF8, "application/json")
        };

        if (!string.IsNullOrWhiteSpace(options.ApiKey))
        {
            request.Headers.Authorization = new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", options.ApiKey);
        }

        using var response = await httpClient.SendAsync(request, cancellationToken);
        var body = await response.Content.ReadAsStringAsync(cancellationToken);

        if (!response.IsSuccessStatusCode)
        {
            throw new InvalidOperationException($"LLM request failed ({(int)response.StatusCode}): {body}");
        }

        using var document = JsonDocument.Parse(body);
        if (!document.RootElement.TryGetProperty("choices", out var choices) || choices.GetArrayLength() == 0)
        {
            throw new InvalidOperationException("LLM returned no choices.");
        }

        var content = choices[0]
            .GetProperty("message")
            .GetProperty("content")
            .GetString();

        if (string.IsNullOrWhiteSpace(content))
        {
            throw new InvalidOperationException("LLM returned an empty response.");
        }

        return content.Trim();
    }
}
