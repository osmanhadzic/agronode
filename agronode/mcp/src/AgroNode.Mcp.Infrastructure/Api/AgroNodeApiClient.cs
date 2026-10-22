using System.Globalization;
using System.Net;
using System.Text.Json;
using System.Text.Json.Nodes;
using AgroNode.Mcp.Application.Devices;
using AgroNode.Mcp.Application.Telemetry;
using AgroNode.Mcp.Infrastructure.Configuration;

namespace AgroNode.Mcp.Infrastructure.Api;

/// <summary>Read-only client for the existing AgroNode Go backend REST API.</summary>
public sealed class AgroNodeApiClient(HttpClient httpClient, AgroNodeOptions options) : IDeviceGateway, ITelemetryGateway
{
    public const string HttpClientName = "agronode-api";

    private static readonly JsonSerializerOptions JsonOptions = new() { WriteIndented = true };

    public async Task<string> GetDevicesAsync(CancellationToken cancellationToken) =>
        Format(await GetAsync("/api/devices", cancellationToken));

    public async Task<string> GetDeviceAsync(string deviceId, CancellationToken cancellationToken) =>
        Format(await GetAsync($"/api/devices/{Uri.EscapeDataString(deviceId)}", cancellationToken));

    public async Task<string> GetLatestAsync(string deviceId, CancellationToken cancellationToken) =>
        Format(await GetAsync($"/api/latest/{Uri.EscapeDataString(deviceId)}", cancellationToken));

    // The backend date filter exists only on GET /api/data/{deviceId} (startDate/endDate, RFC3339);
    // the per-sensor endpoint has no filters, so the range, sensor and limit are applied here.
    public async Task<string> GetHistoryAsync(TelemetryHistoryQuery query, CancellationToken cancellationToken)
    {
        var device = Uri.EscapeDataString(query.DeviceId);
        var path = string.IsNullOrWhiteSpace(query.SensorId)
            ? $"/api/data/{device}?startDate={Uri.EscapeDataString(ToRfc3339(query.From))}&endDate={Uri.EscapeDataString(ToRfc3339(query.To))}"
            : $"/api/data/{device}/{Uri.EscapeDataString(query.SensorId)}";

        var body = await GetAsync(path, cancellationToken);

        if (JsonNode.Parse(body) is not JsonArray readings)
        {
            return Format(body);
        }

        IEnumerable<JsonNode?> selected = readings;

        if (!string.IsNullOrWhiteSpace(query.SensorId))
        {
            selected = selected.Where(reading =>
                DateTimeOffset.TryParse(reading?["createdAt"]?.GetValue<string>(), CultureInfo.InvariantCulture,
                    DateTimeStyles.AssumeUniversal, out var createdAt) &&
                createdAt >= query.From && createdAt <= query.To);
        }

        if (query.Limit is > 0)
        {
            selected = selected.Take(query.Limit.Value);
        }

        var result = new JsonArray();
        foreach (var reading in selected)
        {
            result.Add(reading?.DeepClone());
        }

        return result.ToJsonString(JsonOptions);
    }

    private async Task<string> GetAsync(string pathAndQuery, CancellationToken cancellationToken)
    {
        using var request = new HttpRequestMessage(HttpMethod.Get, pathAndQuery);
        request.Headers.TryAddWithoutValidation("X-Organization-ID", options.OrganizationId);

        using var response = await httpClient.SendAsync(request, cancellationToken);
        var body = await response.Content.ReadAsStringAsync(cancellationToken);

        if (response.StatusCode == HttpStatusCode.NotFound)
        {
            throw new InvalidOperationException("Requested resource was not found in AgroNode backend.");
        }

        if (response.StatusCode == HttpStatusCode.Unauthorized)
        {
            throw new InvalidOperationException(
                "AgroNode backend rejected the request (401). Set AGRONODE_MCP_API_TOKEN or AGRONODE_MCP_LOGIN_EMAIL/AGRONODE_MCP_LOGIN_PASSWORD.");
        }

        if (!response.IsSuccessStatusCode)
        {
            throw new InvalidOperationException($"AgroNode backend returned {(int)response.StatusCode}: {TrimForError(body)}");
        }

        return string.IsNullOrWhiteSpace(body) ? "{}" : body;
    }

    private static string Format(string body)
    {
        using var document = JsonDocument.Parse(body);
        return JsonSerializer.Serialize(document.RootElement, JsonOptions);
    }

    private static string ToRfc3339(DateTimeOffset value) =>
        value.UtcDateTime.ToString("yyyy-MM-dd'T'HH:mm:ss'Z'", CultureInfo.InvariantCulture);

    private static string TrimForError(string body)
    {
        const int maxLength = 400;

        if (string.IsNullOrWhiteSpace(body))
        {
            return "<empty response>";
        }

        return body.Length <= maxLength ? body : body[..maxLength] + "...";
    }
}
