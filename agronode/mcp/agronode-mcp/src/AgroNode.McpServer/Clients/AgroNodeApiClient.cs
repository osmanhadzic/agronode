using System.Net;
using System.Text.Json;
using AgroNode.McpServer.Options;

namespace AgroNode.McpServer.Clients;

internal sealed class AgroNodeApiClient(HttpClient httpClient, AgroNodeOptions options) : IAgroNodeApiClient
{
    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        WriteIndented = true
    };

    public Task<string> GetDevicesAsync(CancellationToken cancellationToken) =>
        GetAsync("/api/devices", cancellationToken);

    public Task<string> GetDeviceStatusAsync(string deviceId, CancellationToken cancellationToken) =>
        GetAsync($"/api/devices/{Uri.EscapeDataString(deviceId)}", cancellationToken);

    public Task<string> GetLatestTelemetryAsync(string deviceId, CancellationToken cancellationToken) =>
        GetAsync($"/api/latest/{Uri.EscapeDataString(deviceId)}", cancellationToken);

    public Task<string> GetTelemetryHistoryAsync(
        string deviceId,
        string from,
        string to,
        string? sensorId,
        int? limit,
        CancellationToken cancellationToken)
    {
        var query = new List<string>
        {
            $"from={Uri.EscapeDataString(from)}",
            $"to={Uri.EscapeDataString(to)}"
        };

        if (!string.IsNullOrWhiteSpace(sensorId))
        {
            query.Add($"sensorId={Uri.EscapeDataString(sensorId)}");
        }

        if (limit is > 0)
        {
            query.Add($"limit={limit.Value}");
        }

        var queryString = string.Join("&", query);
        return GetAsync($"/api/telemetry/{Uri.EscapeDataString(deviceId)}?{queryString}", cancellationToken);
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

        if (!response.IsSuccessStatusCode)
        {
            throw new InvalidOperationException($"AgroNode backend returned {(int)response.StatusCode}: {TrimForError(body)}");
        }

        if (string.IsNullOrWhiteSpace(body))
        {
            return "{}";
        }

        using var jsonDocument = JsonDocument.Parse(body);
        return JsonSerializer.Serialize(jsonDocument.RootElement, JsonOptions);
    }

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
