using System.Text.Json.Nodes;

namespace AgroNode.Mcp.Server.Mcp;

public enum ToolAccess
{
    /// <summary>Only reads data; always allowed.</summary>
    ReadOnly,

    /// <summary>Changes device or system state; requires explicit opt-in (see ToolAccessPolicy).</summary>
    Command
}

public sealed record McpTool(
    string Name,
    string Description,
    JsonObject InputSchema,
    Func<JsonObject?, CancellationToken, Task<string>> Handler,
    ToolAccess Access = ToolAccess.ReadOnly);

public interface IMcpToolProvider
{
    IEnumerable<McpTool> GetTools();
}
