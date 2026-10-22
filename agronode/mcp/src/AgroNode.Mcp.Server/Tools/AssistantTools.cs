using System.Text.Json.Nodes;
using AgroNode.Mcp.Application.Assistant;
using AgroNode.Mcp.Server.Mcp;

namespace AgroNode.Mcp.Server.Tools;

public sealed class AssistantTools(IAssistantService assistant) : IMcpToolProvider
{
    public IEnumerable<McpTool> GetTools()
    {
        yield return new McpTool(
            "ask_llm",
            "Asks configured LLM provider and returns a concise answer.",
            ToolArguments.Schema(
                new JsonObject
                {
                    ["prompt"] = ToolArguments.Property("string", minLength: 1),
                    ["system_prompt"] = ToolArguments.Property("string", "Optional system instruction override.")
                },
                "prompt"),
            (args, ct) => assistant.AskAsync(
                ToolArguments.String(args, "prompt"),
                ToolArguments.String(args, "system_prompt"),
                ct));
    }
}
