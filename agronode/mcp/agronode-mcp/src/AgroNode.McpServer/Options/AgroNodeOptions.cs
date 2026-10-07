namespace AgroNode.McpServer.Options;

internal sealed class AgroNodeOptions
{
    public string RestBaseUrl { get; init; } = "http://localhost:8080";
    public string OrganizationId { get; init; } = "1";
    public int TimeoutSeconds { get; init; } = 15;
}

internal sealed class McpOptions
{
    public string ServerName { get; init; } = "agronode-mcp-dotnet";
    public string ServerVersion { get; init; } = "0.1.0";
    public string ProtocolVersion { get; init; } = "2024-11-05";
}

internal sealed class LlmOptions
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
