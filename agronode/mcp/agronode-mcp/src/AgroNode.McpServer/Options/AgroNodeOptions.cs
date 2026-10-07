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
