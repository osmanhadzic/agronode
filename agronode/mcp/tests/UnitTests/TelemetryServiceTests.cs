using AgroNode.Mcp.Application.Common;
using AgroNode.Mcp.Application.Telemetry;
using AgroNode.Mcp.UnitTests.Fakes;

namespace AgroNode.Mcp.UnitTests;

public class TelemetryServiceTests
{
    private readonly FakeTelemetryGateway gateway = new();
    private readonly TelemetryService service;

    public TelemetryServiceTests() => service = new TelemetryService(gateway);

    [Fact]
    public async Task GetLatest_RequiresDeviceId()
    {
        await Assert.ThrowsAsync<ValidationException>(() => service.GetLatestAsync(" ", CancellationToken.None));

        await service.GetLatestAsync("dev-1", CancellationToken.None);
        Assert.Equal("dev-1", gateway.LatestDeviceId);
    }

    [Fact]
    public async Task GetHistory_ParsesRangeAndForwardsQuery()
    {
        await service.GetHistoryAsync("dev-1", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00+01:00", " temp ", 50, CancellationToken.None);

        var query = gateway.HistoryQuery!;
        Assert.Equal("dev-1", query.DeviceId);
        Assert.Equal(new DateTimeOffset(2026, 1, 1, 0, 0, 0, TimeSpan.Zero), query.From);
        Assert.Equal(new DateTimeOffset(2026, 1, 2, 0, 0, 0, TimeSpan.FromHours(1)), query.To);
        Assert.Equal("temp", query.SensorId);
        Assert.Equal(50, query.Limit);
    }

    [Fact]
    public async Task GetHistory_TreatsBlankSensorAsNone()
    {
        await service.GetHistoryAsync("dev-1", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", " ", null, CancellationToken.None);

        Assert.Null(gateway.HistoryQuery!.SensorId);
        Assert.Null(gateway.HistoryQuery.Limit);
    }

    [Theory]
    [InlineData("not-a-date", "2026-01-02T00:00:00Z")]
    [InlineData("2026-01-01T00:00:00Z", "")]
    [InlineData("2026-01-03T00:00:00Z", "2026-01-02T00:00:00Z")]
    public async Task GetHistory_RejectsInvalidRange(string from, string to)
    {
        await Assert.ThrowsAsync<ValidationException>(
            () => service.GetHistoryAsync("dev-1", from, to, null, null, CancellationToken.None));
        Assert.Null(gateway.HistoryQuery);
    }

    [Theory]
    [InlineData(0)]
    [InlineData(-1)]
    [InlineData(TelemetryService.MaxLimit + 1)]
    public async Task GetHistory_RejectsLimitOutOfRange(int limit)
    {
        await Assert.ThrowsAsync<ValidationException>(
            () => service.GetHistoryAsync("dev-1", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", null, limit, CancellationToken.None));
    }
}
