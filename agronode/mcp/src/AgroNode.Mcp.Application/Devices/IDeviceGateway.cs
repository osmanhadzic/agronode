namespace AgroNode.Mcp.Application.Devices;

/// <summary>Read-only access to device data in the AgroNode backend. Results are JSON documents.</summary>
public interface IDeviceGateway
{
    Task<string> GetDevicesAsync(CancellationToken cancellationToken);

    Task<string> GetDeviceAsync(string deviceId, CancellationToken cancellationToken);
}
