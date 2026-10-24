using AgroNode.Mcp.Server.Mcp;

namespace AgroNode.Mcp.Server.Security;

public sealed class SecurityOptions
{
    /// <summary>Must be enabled explicitly before any tool that changes device state can be listed or called.</summary>
    public bool AllowCommands { get; init; }

    public static SecurityOptions FromEnvironment(Func<string, string?>? environment = null)
    {
        environment ??= Environment.GetEnvironmentVariable;
        return new SecurityOptions
        {
            AllowCommands = bool.TryParse(environment("AGRONODE_MCP_ALLOW_COMMANDS"), out var allow) && allow
        };
    }
}

/// <summary>Keeps command (state-changing) tools unavailable unless explicitly enabled.</summary>
public sealed class ToolAccessPolicy(SecurityOptions options)
{
    public bool IsAllowed(McpTool tool) => tool.Access == ToolAccess.ReadOnly || options.AllowCommands;
}
