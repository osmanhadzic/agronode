namespace AgroNode.Mcp.Application.Devices;

public interface IDeviceService
{
    Task<string> GetDevicesAsync(CancellationToken cancellationToken);

    Task<string> GetDeviceStatusAsync(string? deviceId, CancellationToken cancellationToken);
}
