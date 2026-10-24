using AgroNode.Mcp.Infrastructure.Configuration;
using AgroNode.Mcp.Server;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Http;

namespace AgroNode.Mcp.IntegrationTests;

internal static class TestHost
{
    public static ServiceProvider Build(StubBackendHandler backend, AgroNodeOptions? agroNode = null)
    {
        var config = new AppConfiguration
        {
            AgroNode = agroNode ?? new AgroNodeOptions { RestBaseUrl = "http://backend.test", OrganizationId = "7" },
            Mcp = new McpOptions(),
            Llm = new LlmOptions()
        };

        var services = new ServiceCollection();
        services.AddAgroNodeServer(config);
        services.ConfigureAll<HttpClientFactoryOptions>(options =>
            options.HttpMessageHandlerBuilderActions.Add(builder => builder.PrimaryHandler = backend));

        return services.BuildServiceProvider(new ServiceProviderOptions { ValidateOnBuild = true, ValidateScopes = true });
    }
}
