using AgroNode.McpServer.Clients;
using AgroNode.McpServer.Options;
using Microsoft.Extensions.DependencyInjection;

namespace AgroNode.McpServer;

internal static class DependencyInjection
{
    public static IServiceCollection AddAgroNodeMcp(this IServiceCollection services, AppConfiguration config)
    {
        services.AddSingleton(config.AgroNode);
        services.AddSingleton(config.Mcp);

        services.AddSingleton(sp =>
        {
            var options = sp.GetRequiredService<AgroNodeOptions>();
            return new HttpClient
            {
                BaseAddress = new Uri(options.RestBaseUrl),
                Timeout = TimeSpan.FromSeconds(Math.Max(1, options.TimeoutSeconds))
            };
        });

        services.AddSingleton<IAgroNodeApiClient, AgroNodeApiClient>();
        services.AddSingleton<McpStdioServer>();

        return services;
    }
}
