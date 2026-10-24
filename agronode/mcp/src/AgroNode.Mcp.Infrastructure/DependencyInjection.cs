using AgroNode.Mcp.Application.Assistant;
using AgroNode.Mcp.Application.Devices;
using AgroNode.Mcp.Application.Telemetry;
using AgroNode.Mcp.Infrastructure.Api;
using AgroNode.Mcp.Infrastructure.Configuration;
using AgroNode.Mcp.Infrastructure.Llm;
using Microsoft.Extensions.DependencyInjection;

namespace AgroNode.Mcp.Infrastructure;

public static class DependencyInjection
{
    public const string LlmHttpClientName = "agronode-llm";

    public static IServiceCollection AddAgroNodeInfrastructure(this IServiceCollection services, AppConfiguration config)
    {
        services.AddSingleton(config.AgroNode);
        services.AddSingleton(config.Mcp);
        services.AddSingleton(config.Llm);
        services.AddSingleton(TimeProvider.System);

        services.AddHttpClient(AgroNodeTokenProvider.AuthClientName, (sp, client) => ConfigureBackend(client, config.AgroNode));
        services.AddSingleton<IAgroNodeTokenProvider, AgroNodeTokenProvider>();
        services.AddTransient<AgroNodeAuthHandler>();

        services.AddHttpClient<AgroNodeApiClient>(AgroNodeApiClient.HttpClientName, client => ConfigureBackend(client, config.AgroNode))
            .AddHttpMessageHandler<AgroNodeAuthHandler>();
        services.AddTransient<IDeviceGateway>(sp => sp.GetRequiredService<AgroNodeApiClient>());
        services.AddTransient<ITelemetryGateway>(sp => sp.GetRequiredService<AgroNodeApiClient>());

        services.AddHttpClient(LlmHttpClientName, client =>
        {
            client.BaseAddress = new Uri(config.Llm.BaseUrl);
            client.Timeout = TimeSpan.FromSeconds(Math.Max(1, config.Llm.TimeoutSeconds));
        });
        services.AddTransient<ILlmClient>(sp =>
        {
            var options = sp.GetRequiredService<LlmOptions>();
            if (!options.Enabled)
            {
                return new NoopLlmClient();
            }

            var httpClient = sp.GetRequiredService<IHttpClientFactory>().CreateClient(LlmHttpClientName);
            return options.Provider.Trim().ToLowerInvariant() switch
            {
                "openai" or "openai-compatible" or "external" => new OpenAiCompatibleLlmClient(httpClient, options),
                _ => new OllamaLlmClient(httpClient, options)
            };
        });

        return services;
    }

    private static void ConfigureBackend(HttpClient client, AgroNodeOptions options)
    {
        client.BaseAddress = new Uri(options.RestBaseUrl);
        client.Timeout = TimeSpan.FromSeconds(Math.Max(1, options.TimeoutSeconds));
    }
}
