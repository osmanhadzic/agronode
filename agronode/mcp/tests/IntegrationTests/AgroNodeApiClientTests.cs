using System.Net;
using System.Text.Json.Nodes;
using AgroNode.Mcp.Application.Devices;
using AgroNode.Mcp.Application.Telemetry;
using AgroNode.Mcp.Infrastructure.Configuration;
using Microsoft.Extensions.DependencyInjection;

namespace AgroNode.Mcp.IntegrationTests;

public class AgroNodeApiClientTests
{
    [Fact]
    public async Task Devices_UseExistingBackendRoutesAndOrganizationHeader()
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json("[{\"deviceId\":\"d1\"}]"));
        await using var provider = TestHost.Build(backend);
        var devices = provider.GetRequiredService<IDeviceGateway>();

        await devices.GetDevicesAsync(CancellationToken.None);
        await devices.GetDeviceAsync("dev 1", CancellationToken.None);

        Assert.Equal("/api/devices", backend.Requests[0].RequestUri!.AbsolutePath);
        Assert.Equal("/api/devices/dev%201", backend.Requests[1].RequestUri!.AbsolutePath);
        Assert.Equal("7", backend.Requests[0].Headers.GetValues("X-Organization-ID").Single());
        Assert.All(backend.Requests, request => Assert.Equal(HttpMethod.Get, request.Method));
    }

    [Fact]
    public async Task Latest_CallsLatestRoute()
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json("{\"deviceId\":\"d1\"}"));
        await using var provider = TestHost.Build(backend);

        await provider.GetRequiredService<ITelemetryGateway>().GetLatestAsync("d1", CancellationToken.None);

        Assert.Equal("/api/latest/d1", backend.Requests.Single().RequestUri!.AbsolutePath);
    }

    [Fact]
    public async Task History_UsesDateFilteredDataRouteAndAppliesLimit()
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json(
            "[{\"deviceId\":\"d1\",\"createdAt\":\"2026-01-01T01:00:00Z\"},{\"deviceId\":\"d1\",\"createdAt\":\"2026-01-01T02:00:00Z\"},{\"deviceId\":\"d1\",\"createdAt\":\"2026-01-01T03:00:00Z\"}]"));
        await using var provider = TestHost.Build(backend);
        var query = new TelemetryHistoryQuery(
            "d1", new DateTimeOffset(2026, 1, 1, 0, 0, 0, TimeSpan.Zero), new DateTimeOffset(2026, 1, 2, 0, 0, 0, TimeSpan.Zero), null, 2);

        var result = await provider.GetRequiredService<ITelemetryGateway>().GetHistoryAsync(query, CancellationToken.None);

        var uri = backend.Requests.Single().RequestUri!;
        Assert.Equal("/api/data/d1", uri.AbsolutePath);
        Assert.Equal("?startDate=2026-01-01T00%3A00%3A00Z&endDate=2026-01-02T00%3A00%3A00Z", uri.Query);
        Assert.Equal(2, JsonNode.Parse(result)!.AsArray().Count);
    }

    [Fact]
    public async Task History_WithSensor_UsesSensorRouteAndFiltersByRange()
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json(
            "[{\"createdAt\":\"2025-12-31T23:00:00Z\"},{\"createdAt\":\"2026-01-01T05:00:00Z\"}]"));
        await using var provider = TestHost.Build(backend);
        var query = new TelemetryHistoryQuery(
            "d1", new DateTimeOffset(2026, 1, 1, 0, 0, 0, TimeSpan.Zero), new DateTimeOffset(2026, 1, 2, 0, 0, 0, TimeSpan.Zero), "temp", null);

        var result = await provider.GetRequiredService<ITelemetryGateway>().GetHistoryAsync(query, CancellationToken.None);

        Assert.Equal("/api/data/d1/temp", backend.Requests.Single().RequestUri!.AbsolutePath);
        Assert.Single(JsonNode.Parse(result)!.AsArray());
    }

    [Theory]
    [InlineData(HttpStatusCode.NotFound, "not found")]
    [InlineData(HttpStatusCode.Unauthorized, "401")]
    [InlineData(HttpStatusCode.InternalServerError, "500")]
    public async Task ErrorStatuses_AreSurfacedAsFailures(HttpStatusCode status, string expectedText)
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json("{\"error\":\"x\"}", status));
        await using var provider = TestHost.Build(backend);

        var exception = await Assert.ThrowsAsync<InvalidOperationException>(
            () => provider.GetRequiredService<IDeviceGateway>().GetDevicesAsync(CancellationToken.None));

        Assert.Contains(expectedText, exception.Message, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public async Task StaticToken_IsSentAsBearer()
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json("[]"));
        await using var provider = TestHost.Build(backend, new AgroNodeOptions { RestBaseUrl = "http://backend.test", ApiToken = "abc" });

        await provider.GetRequiredService<IDeviceGateway>().GetDevicesAsync(CancellationToken.None);

        Assert.Equal("Bearer abc", backend.Requests.Single().Headers.Authorization!.ToString());
    }

    [Fact]
    public async Task LoginCredentials_ObtainTokenOnceAndReuseIt()
    {
        var expiresAt = DateTimeOffset.UtcNow.AddHours(1).ToUnixTimeSeconds();
        var backend = new StubBackendHandler(request => request.RequestUri!.AbsolutePath == "/api/auth/login"
            ? StubBackendHandler.Json($"{{\"token\":\"session-1\",\"expiresAt\":{expiresAt}}}")
            : StubBackendHandler.Json("[]"));
        await using var provider = TestHost.Build(
            backend,
            new AgroNodeOptions { RestBaseUrl = "http://backend.test", LoginEmail = "a@b.c", LoginPassword = "pw" });
        var devices = provider.GetRequiredService<IDeviceGateway>();

        await devices.GetDevicesAsync(CancellationToken.None);
        await devices.GetDevicesAsync(CancellationToken.None);

        Assert.Single(backend.Requests, request => request.RequestUri!.AbsolutePath == "/api/auth/login");
        var apiCalls = backend.Requests.Where(request => request.RequestUri!.AbsolutePath == "/api/devices").ToList();
        Assert.Equal(2, apiCalls.Count);
        Assert.All(apiCalls, request => Assert.Equal("Bearer session-1", request.Headers.Authorization!.ToString()));
    }
}
