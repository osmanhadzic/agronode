using System.Text.Json.Nodes;
using AgroNode.McpServer.Options;

namespace AgroNode.McpServer;

internal sealed class AppConfiguration
{
    public required AgroNodeOptions AgroNode { get; init; }
    public required McpOptions Mcp { get; init; }
}

internal static class ConfigurationLoader
{
    public static AppConfiguration Load()
    {
        var appSettingsPath = Path.Combine(AppContext.BaseDirectory, "appsettings.json");
        var appSettings = File.Exists(appSettingsPath)
            ? JsonNode.Parse(File.ReadAllText(appSettingsPath))
            : new JsonObject();

        var agroNodeObject = appSettings?["AgroNode"] as JsonObject ?? new JsonObject();
        var mcpObject = appSettings?["Mcp"] as JsonObject ?? new JsonObject();

        var agroNode = new AgroNodeOptions
        {
            RestBaseUrl = GetValue("AGRONODE_MCP_REST_BASE_URL", agroNodeObject["RestBaseUrl"]?.GetValue<string>(), "http://localhost:8080"),
            OrganizationId = GetValue("AGRONODE_MCP_ORGANIZATION_ID", agroNodeObject["OrganizationId"]?.GetValue<string>(), "1"),
            TimeoutSeconds = ParseInt(GetValue("AGRONODE_MCP_TIMEOUT_SECONDS", agroNodeObject["TimeoutSeconds"]?.ToString(), "15"), 15)
        };

        var mcp = new McpOptions
        {
            ServerName = GetValue("AGRONODE_MCP_SERVER_NAME", mcpObject["ServerName"]?.GetValue<string>(), "agronode-mcp-dotnet"),
            ServerVersion = GetValue("AGRONODE_MCP_SERVER_VERSION", mcpObject["ServerVersion"]?.GetValue<string>(), "0.1.0"),
            ProtocolVersion = GetValue("AGRONODE_MCP_PROTOCOL_VERSION", mcpObject["ProtocolVersion"]?.GetValue<string>(), "2024-11-05")
        };

        return new AppConfiguration
        {
            AgroNode = agroNode,
            Mcp = mcp
        };
    }

    private static string GetValue(string envName, string? configValue, string fallback)
    {
        var envValue = Environment.GetEnvironmentVariable(envName);
        if (!string.IsNullOrWhiteSpace(envValue))
        {
            return envValue;
        }

        return string.IsNullOrWhiteSpace(configValue) ? fallback : configValue;
    }

    private static int ParseInt(string value, int fallback) => int.TryParse(value, out var parsed) ? parsed : fallback;
}
