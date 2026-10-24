using System.Text.Json.Nodes;

namespace AgroNode.Mcp.Infrastructure.Configuration;

public static class ConfigurationLoader
{
    public static AppConfiguration Load(Func<string, string?>? environment = null, string? appSettingsPath = null)
    {
        environment ??= Environment.GetEnvironmentVariable;
        appSettingsPath ??= Path.Combine(AppContext.BaseDirectory, "appsettings.json");

        var appSettings = File.Exists(appSettingsPath)
            ? JsonNode.Parse(File.ReadAllText(appSettingsPath))
            : new JsonObject();

        var agroNodeObject = appSettings?["AgroNode"] as JsonObject ?? new JsonObject();
        var mcpObject = appSettings?["Mcp"] as JsonObject ?? new JsonObject();
        var llmObject = appSettings?["Llm"] as JsonObject ?? new JsonObject();

        string Get(string envName, JsonObject section, string key, string fallback) =>
            GetValue(environment, envName, section[key]?.ToString(), fallback);

        // Secrets (token, password, API key) are never read from appsettings.json defaults.
        var agroNode = new AgroNodeOptions
        {
            RestBaseUrl = Get("AGRONODE_MCP_REST_BASE_URL", agroNodeObject, "RestBaseUrl", "http://localhost:8080"),
            OrganizationId = Get("AGRONODE_MCP_ORGANIZATION_ID", agroNodeObject, "OrganizationId", "1"),
            TimeoutSeconds = ParseInt(Get("AGRONODE_MCP_TIMEOUT_SECONDS", agroNodeObject, "TimeoutSeconds", "15"), 15),
            ApiToken = GetValue(environment, "AGRONODE_MCP_API_TOKEN", null, string.Empty),
            LoginEmail = GetValue(environment, "AGRONODE_MCP_LOGIN_EMAIL", null, string.Empty),
            LoginPassword = GetValue(environment, "AGRONODE_MCP_LOGIN_PASSWORD", null, string.Empty)
        };

        var mcp = new McpOptions
        {
            ServerName = Get("AGRONODE_MCP_SERVER_NAME", mcpObject, "ServerName", "agronode-mcp-dotnet"),
            ServerVersion = Get("AGRONODE_MCP_SERVER_VERSION", mcpObject, "ServerVersion", "0.1.0"),
            ProtocolVersion = Get("AGRONODE_MCP_PROTOCOL_VERSION", mcpObject, "ProtocolVersion", "2024-11-05")
        };

        var llm = new LlmOptions
        {
            Enabled = ParseBool(Get("AGRONODE_MCP_LLM_ENABLED", llmObject, "Enabled", "false"), false),
            Provider = Get("AGRONODE_MCP_LLM_PROVIDER", llmObject, "Provider", "ollama"),
            BaseUrl = Get("AGRONODE_MCP_LLM_BASE_URL", llmObject, "BaseUrl", "http://localhost:11434"),
            ApiKey = GetValue(environment, "AGRONODE_MCP_LLM_API_KEY", null, string.Empty),
            Model = Get("AGRONODE_MCP_LLM_MODEL", llmObject, "Model", "llama3.2:1b"),
            ChatCompletionsPath = Get("AGRONODE_MCP_LLM_CHAT_COMPLETIONS_PATH", llmObject, "ChatCompletionsPath", "/v1/chat/completions"),
            SystemPrompt = Get(
                "AGRONODE_MCP_LLM_SYSTEM_PROMPT",
                llmObject,
                "SystemPrompt",
                "You are AgroNode MCP assistant. Provide concise, practical answers for IoT operations."),
            TimeoutSeconds = ParseInt(Get("AGRONODE_MCP_LLM_TIMEOUT_SECONDS", llmObject, "TimeoutSeconds", "20"), 20)
        };

        return new AppConfiguration { AgroNode = agroNode, Mcp = mcp, Llm = llm };
    }

    private static string GetValue(Func<string, string?> environment, string envName, string? configValue, string fallback)
    {
        var envValue = environment(envName);
        if (!string.IsNullOrWhiteSpace(envValue))
        {
            return envValue;
        }

        return string.IsNullOrWhiteSpace(configValue) ? fallback : configValue;
    }

    private static int ParseInt(string value, int fallback) =>
        int.TryParse(value, out var parsed) && parsed > 0 ? parsed : fallback;

    private static bool ParseBool(string value, bool fallback) =>
        bool.TryParse(value, out var parsed) ? parsed : fallback;
}
