using AgroNode.Mcp.Application;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Hosting;
using AgroNode.Mcp.Infrastructure;
using AgroNode.Mcp.Infrastructure.Configuration;
using AgroNode.Mcp.Server.Mcp;
using AgroNode.Mcp.Server.Security;
using AgroNode.Mcp.Server.Tools;
using Microsoft.Extensions.DependencyInjection;

namespace AgroNode.Mcp.Server;

public static class Program
{
    public static async Task Main()
    {
        var config = ConfigurationLoader.Load();

        if (string.Equals(Environment.GetEnvironmentVariable("AGRONODE_MCP_TRANSPORT"), "http", StringComparison.OrdinalIgnoreCase))
        {
            await RunHttpAsync(config);
            return;
        }

        var services = new ServiceCollection();
        services.AddAgroNodeServer(config);

        using var provider = services.BuildServiceProvider(new ServiceProviderOptions
        {
            ValidateOnBuild = true,
            ValidateScopes = true
        });

        var server = provider.GetRequiredService<McpStdioServer>();

        await server.RunAsync(Console.OpenStandardInput(), Console.OpenStandardOutput(), CancellationToken.None);
    }

    private static async Task RunHttpAsync(AppConfiguration config)
    {
        var port = Environment.GetEnvironmentVariable("AGRONODE_MCP_HTTP_PORT") ?? "8090";
        var host = Environment.GetEnvironmentVariable("AGRONODE_MCP_HTTP_HOST") ?? "127.0.0.1";
        var token = Environment.GetEnvironmentVariable("AGRONODE_MCP_HTTP_TOKEN");

        var builder = WebApplication.CreateSlimBuilder();
        builder.WebHost.UseUrls($"http://{host}:{port}");
        builder.Services.AddAgroNodeServer(config);

        var app = builder.Build();
        var server = app.Services.GetRequiredService<McpStdioServer>();

        app.MapGet("/health", () => Results.Ok(new { status = "ok" }));

        app.MapPost("/mcp", async (HttpContext context) =>
        {
            if (!string.IsNullOrEmpty(token) && context.Request.Headers.Authorization != $"Bearer {token}")
            {
                return Results.Unauthorized();
            }

            using var reader = new StreamReader(context.Request.Body);
            var payload = await reader.ReadToEndAsync(context.RequestAborted);
            var response = await server.HandleRequestAsync(payload, context.RequestAborted);

            return response is null
                ? Results.Accepted()
                : Results.Content(response, "application/json");
        });

        await app.RunAsync();
    }

    public static IServiceCollection AddAgroNodeServer(this IServiceCollection services, AppConfiguration config)
    {
        services.AddAgroNodeApplication();
        services.AddAgroNodeInfrastructure(config);

        services.AddSingleton(SecurityOptions.FromEnvironment());
        services.AddSingleton<ToolAccessPolicy>();

        services.AddTransient<IMcpToolProvider, DeviceTools>();
        services.AddTransient<IMcpToolProvider, TelemetryTools>();
        services.AddTransient<IMcpToolProvider, AssistantTools>();
        services.AddSingleton<McpStdioServer>();

        return services;
    }
}
