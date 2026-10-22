using AgroNode.Mcp.Application.Common;

namespace AgroNode.Mcp.Application.Devices;

public sealed class DeviceService(IDeviceGateway gateway) : IDeviceService
{
    public Task<string> GetDevicesAsync(CancellationToken cancellationToken) =>
        gateway.GetDevicesAsync(cancellationToken);

    public Task<string> GetDeviceStatusAsync(string? deviceId, CancellationToken cancellationToken) =>
        gateway.GetDeviceAsync(Guard.RequireText(deviceId, "device_id"), cancellationToken);
}
