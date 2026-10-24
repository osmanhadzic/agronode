using AgroNode.Mcp.Application.Assistant;
using AgroNode.Mcp.Application.Devices;
using AgroNode.Mcp.Application.Telemetry;
using Microsoft.Extensions.DependencyInjection;

namespace AgroNode.Mcp.Application;

public static class DependencyInjection
{
    public static IServiceCollection AddAgroNodeApplication(this IServiceCollection services)
    {
        services.AddTransient<IDeviceService, DeviceService>();
        services.AddTransient<ITelemetryService, TelemetryService>();
        services.AddTransient<IAssistantService, AssistantService>();

        return services;
    }
}
