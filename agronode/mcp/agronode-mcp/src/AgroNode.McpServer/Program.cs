using Microsoft.Extensions.DependencyInjection;

namespace AgroNode.McpServer;

internal static class Program
{
    public static async Task Main()
    {
        var config = ConfigurationLoader.Load();

        var services = new ServiceCollection();
        services.AddAgroNodeMcp(config);

        using var provider = services.BuildServiceProvider(new ServiceProviderOptions
        {
            ValidateOnBuild = true,
            ValidateScopes = true
        });

        var server = provider.GetRequiredService<McpStdioServer>();

        await server.RunAsync(Console.OpenStandardInput(), Console.OpenStandardOutput(), CancellationToken.None);
    }
}
