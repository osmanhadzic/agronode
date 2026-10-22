namespace AgroNode.Mcp.Application.Telemetry;

/// <summary>Read-only access to telemetry in the AgroNode backend. Results are JSON documents.</summary>
public interface ITelemetryGateway
{
    Task<string> GetLatestAsync(string deviceId, CancellationToken cancellationToken);

    Task<string> GetHistoryAsync(TelemetryHistoryQuery query, CancellationToken cancellationToken);
}
