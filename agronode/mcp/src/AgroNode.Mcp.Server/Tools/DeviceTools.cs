using AgroNode.Mcp.Application.Devices;
using AgroNode.Mcp.Server.Mcp;
using System.Text.Json.Nodes;

namespace AgroNode.Mcp.Server.Tools;

public sealed class DeviceTools(IDeviceService devices) : IMcpToolProvider
{
    public IEnumerable<McpTool> GetTools()
    {
        yield return new McpTool(
            "get_devices",
            "Returns AgroNode devices from backend API.",
            ToolArguments.Schema(new JsonObject()),
            (_, ct) => devices.GetDevicesAsync(ct));

        yield return new McpTool(
            "get_device_status",
            "Returns detailed status for a single device.",
            ToolArguments.Schema(
                new JsonObject { ["device_id"] = ToolArguments.Property("string", minLength: 1) },
                "device_id"),
            (args, ct) => devices.GetDeviceStatusAsync(ToolArguments.String(args, "device_id"), ct));
    }
}
