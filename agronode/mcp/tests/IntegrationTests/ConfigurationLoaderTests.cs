using AgroNode.Mcp.Infrastructure.Configuration;

namespace AgroNode.Mcp.IntegrationTests;

public class ConfigurationLoaderTests
{
    [Fact]
    public void Load_UsesDefaultsWhenNothingConfigured()
    {
        var config = ConfigurationLoader.Load(_ => null, "/nonexistent/appsettings.json");

        Assert.Equal("http://localhost:8080", config.AgroNode.RestBaseUrl);
        Assert.Equal("1", config.AgroNode.OrganizationId);
        Assert.Equal(15, config.AgroNode.TimeoutSeconds);
        Assert.Equal(string.Empty, config.AgroNode.ApiToken);
        Assert.False(config.Llm.Enabled);
    }

    [Fact]
    public void Load_EnvironmentOverridesAndReadsSecrets()
    {
        var env = new Dictionary<string, string>
        {
            ["AGRONODE_MCP_REST_BASE_URL"] = "http://backend:8080",
            ["AGRONODE_MCP_API_TOKEN"] = "token-1",
            ["AGRONODE_MCP_LOGIN_EMAIL"] = "a@b.c",
            ["AGRONODE_MCP_LOGIN_PASSWORD"] = "pw",
            ["AGRONODE_MCP_LLM_ENABLED"] = "true",
            ["AGRONODE_MCP_TIMEOUT_SECONDS"] = "not-a-number"
        };

        var config = ConfigurationLoader.Load(name => env.GetValueOrDefault(name), "/nonexistent/appsettings.json");

        Assert.Equal("http://backend:8080", config.AgroNode.RestBaseUrl);
        Assert.Equal("token-1", config.AgroNode.ApiToken);
        Assert.Equal("a@b.c", config.AgroNode.LoginEmail);
        Assert.Equal("pw", config.AgroNode.LoginPassword);
        Assert.True(config.Llm.Enabled);
        Assert.Equal(15, config.AgroNode.TimeoutSeconds);
    }
}
