using AgroNode.Mcp.Application.Assistant;
using AgroNode.Mcp.Application.Devices;
using AgroNode.Mcp.Application.Telemetry;

namespace AgroNode.Mcp.UnitTests.Fakes;

internal sealed class FakeDeviceGateway : IDeviceGateway
{
    public List<string> RequestedDeviceIds { get; } = [];

    public Task<string> GetDevicesAsync(CancellationToken cancellationToken) => Task.FromResult("[]");

    public Task<string> GetDeviceAsync(string deviceId, CancellationToken cancellationToken)
    {
        RequestedDeviceIds.Add(deviceId);
        return Task.FromResult($"{{\"deviceId\":\"{deviceId}\"}}");
    }
}

internal sealed class FakeTelemetryGateway : ITelemetryGateway
{
    public string? LatestDeviceId { get; private set; }
    public TelemetryHistoryQuery? HistoryQuery { get; private set; }

    public Task<string> GetLatestAsync(string deviceId, CancellationToken cancellationToken)
    {
        LatestDeviceId = deviceId;
        return Task.FromResult("{}");
    }

    public Task<string> GetHistoryAsync(TelemetryHistoryQuery query, CancellationToken cancellationToken)
    {
        HistoryQuery = query;
        return Task.FromResult("[]");
    }
}

internal sealed class FakeLlmClient(bool enabled) : ILlmClient
{
    public bool IsEnabled => enabled;
    public string? Prompt { get; private set; }
    public string? SystemPrompt { get; private set; }

    public Task<string> AskAsync(string prompt, string? systemPromptOverride, CancellationToken cancellationToken)
    {
        Prompt = prompt;
        SystemPrompt = systemPromptOverride;
        return Task.FromResult("answer");
    }
}
