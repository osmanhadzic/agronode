namespace AgroNode.Mcp.Infrastructure.Configuration;

public sealed class AgroNodeOptions
{
    public string RestBaseUrl { get; init; } = "http://localhost:8080";
    public string OrganizationId { get; init; } = "1";
    public int TimeoutSeconds { get; init; } = 15;

    /// <summary>Pre-issued backend session token. Takes precedence over login credentials.</summary>
    public string ApiToken { get; init; } = string.Empty;

    /// <summary>Credentials used to obtain a session token from the backend login endpoint.</summary>
    public string LoginEmail { get; init; } = string.Empty;
    public string LoginPassword { get; init; } = string.Empty;
}

public sealed class McpOptions
{
    public string ServerName { get; init; } = "agronode-mcp-dotnet";
    public string ServerVersion { get; init; } = "0.1.0";
    public string ProtocolVersion { get; init; } = "2024-11-05";
}

public sealed class LlmOptions
{
    public bool Enabled { get; init; }
    public string Provider { get; init; } = "ollama";
    public string BaseUrl { get; init; } = "http://localhost:11434";
    public string ApiKey { get; init; } = string.Empty;
    public string Model { get; init; } = "llama3.2:1b";
    public string ChatCompletionsPath { get; init; } = "/v1/chat/completions";
    public string SystemPrompt { get; init; } = "You are AgroNode MCP assistant. Provide concise, practical answers for IoT operations.";
    public int TimeoutSeconds { get; init; } = 20;
}

public sealed class AppConfiguration
{
    public required AgroNodeOptions AgroNode { get; init; }
    public required McpOptions Mcp { get; init; }
    public required LlmOptions Llm { get; init; }
}
