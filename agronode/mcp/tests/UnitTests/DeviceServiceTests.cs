using AgroNode.Mcp.Application.Common;
using AgroNode.Mcp.Application.Devices;
using AgroNode.Mcp.UnitTests.Fakes;

namespace AgroNode.Mcp.UnitTests;

public class DeviceServiceTests
{
    [Fact]
    public async Task GetDeviceStatus_TrimsIdAndCallsGateway()
    {
        var gateway = new FakeDeviceGateway();
        var service = new DeviceService(gateway);

        var result = await service.GetDeviceStatusAsync("  esp32-01 ", CancellationToken.None);

        Assert.Equal(["esp32-01"], gateway.RequestedDeviceIds);
        Assert.Contains("esp32-01", result);
    }

    [Theory]
    [InlineData(null)]
    [InlineData("")]
    [InlineData("   ")]
    public async Task GetDeviceStatus_RejectsMissingId(string? deviceId)
    {
        var gateway = new FakeDeviceGateway();
        var service = new DeviceService(gateway);

        await Assert.ThrowsAsync<ValidationException>(() => service.GetDeviceStatusAsync(deviceId, CancellationToken.None));
        Assert.Empty(gateway.RequestedDeviceIds);
    }

    [Fact]
    public async Task GetDevices_ReturnsGatewayResult()
    {
        var service = new DeviceService(new FakeDeviceGateway());

        Assert.Equal("[]", await service.GetDevicesAsync(CancellationToken.None));
    }
}
