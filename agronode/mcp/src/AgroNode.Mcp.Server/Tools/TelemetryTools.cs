using System.Text.Json.Nodes;
using AgroNode.Mcp.Application.Telemetry;
using AgroNode.Mcp.Server.Mcp;

namespace AgroNode.Mcp.Server.Tools;

public sealed class TelemetryTools(ITelemetryService telemetry) : IMcpToolProvider
{
    public IEnumerable<McpTool> GetTools()
    {
        yield return new McpTool(
            "get_latest_telemetry",
            "Returns latest telemetry snapshot for a device.",
            ToolArguments.Schema(
                new JsonObject { ["device_id"] = ToolArguments.Property("string", minLength: 1) },
                "device_id"),
            (args, ct) => telemetry.GetLatestAsync(ToolArguments.String(args, "device_id"), ct));

        yield return new McpTool(
            "get_telemetry_history",
            "Returns telemetry history for a device in a time range.",
            ToolArguments.Schema(
                new JsonObject
                {
                    ["device_id"] = ToolArguments.Property("string", minLength: 1),
                    ["from"] = ToolArguments.Property("string", "ISO-8601 timestamp"),
                    ["to"] = ToolArguments.Property("string", "ISO-8601 timestamp"),
                    ["sensor_id"] = ToolArguments.Property("string"),
                    ["limit"] = ToolArguments.Property("integer", minimum: 1, maximum: TelemetryService.MaxLimit)
                },
                "device_id", "from", "to"),
            (args, ct) => telemetry.GetHistoryAsync(
                ToolArguments.String(args, "device_id"),
                ToolArguments.String(args, "from"),
                ToolArguments.String(args, "to"),
                ToolArguments.String(args, "sensor_id"),
                ToolArguments.Int(args, "limit"),
                ct));
    }
}
