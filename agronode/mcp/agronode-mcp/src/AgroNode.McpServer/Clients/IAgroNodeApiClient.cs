namespace AgroNode.McpServer.Clients;

internal interface IAgroNodeApiClient
{
    Task<string> GetDevicesAsync(CancellationToken cancellationToken);

    Task<string> GetDeviceStatusAsync(string deviceId, CancellationToken cancellationToken);

    Task<string> GetLatestTelemetryAsync(string deviceId, CancellationToken cancellationToken);

    Task<string> GetTelemetryHistoryAsync(
        string deviceId,
        string from,
        string to,
        string? sensorId,
        int? limit,
        CancellationToken cancellationToken);
}
