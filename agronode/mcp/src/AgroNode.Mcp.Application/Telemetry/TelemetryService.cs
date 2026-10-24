using AgroNode.Mcp.Application.Common;

namespace AgroNode.Mcp.Application.Telemetry;

public sealed class TelemetryService(ITelemetryGateway gateway) : ITelemetryService
{
    public const int MaxLimit = 5000;

    public Task<string> GetLatestAsync(string? deviceId, CancellationToken cancellationToken) =>
        gateway.GetLatestAsync(Guard.RequireText(deviceId, "device_id"), cancellationToken);

    public Task<string> GetHistoryAsync(
        string? deviceId,
        string? from,
        string? to,
        string? sensorId,
        int? limit,
        CancellationToken cancellationToken)
    {
        var fromValue = ParseTimestamp(from, "from");
        var toValue = ParseTimestamp(to, "to");

        if (fromValue > toValue)
        {
            throw new ValidationException("Argument 'from' must not be later than 'to'.");
        }

        if (limit is < 1 or > MaxLimit)
        {
            throw new ValidationException($"Argument 'limit' must be between 1 and {MaxLimit}.");
        }

        var query = new TelemetryHistoryQuery(
            Guard.RequireText(deviceId, "device_id"),
            fromValue,
            toValue,
            string.IsNullOrWhiteSpace(sensorId) ? null : sensorId.Trim(),
            limit);

        return gateway.GetHistoryAsync(query, cancellationToken);
    }

    private static DateTimeOffset ParseTimestamp(string? value, string field)
    {
        var text = Guard.RequireText(value, field);

        if (!DateTimeOffset.TryParse(text, System.Globalization.CultureInfo.InvariantCulture,
                System.Globalization.DateTimeStyles.AssumeUniversal, out var parsed))
        {
            throw new ValidationException($"Argument '{field}' must be a valid ISO-8601 timestamp.");
        }

        return parsed;
    }
}
