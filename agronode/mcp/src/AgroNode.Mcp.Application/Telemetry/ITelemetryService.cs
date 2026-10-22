namespace AgroNode.Mcp.Application.Telemetry;

public interface ITelemetryService
{
    Task<string> GetLatestAsync(string? deviceId, CancellationToken cancellationToken);

    Task<string> GetHistoryAsync(
        string? deviceId,
        string? from,
        string? to,
        string? sensorId,
        int? limit,
        CancellationToken cancellationToken);
}
