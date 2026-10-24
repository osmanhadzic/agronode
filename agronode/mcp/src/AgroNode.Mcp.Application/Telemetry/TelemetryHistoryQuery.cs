namespace AgroNode.Mcp.Application.Telemetry;

public sealed record TelemetryHistoryQuery(
    string DeviceId,
    DateTimeOffset From,
    DateTimeOffset To,
    string? SensorId,
    int? Limit);
