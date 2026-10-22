using AgroNode.Mcp.Application.Common;

namespace AgroNode.Mcp.Application.Assistant;

public interface IAssistantService
{
    Task<string> AskAsync(string? prompt, string? systemPrompt, CancellationToken cancellationToken);
}

public sealed class AssistantService(ILlmClient llmClient) : IAssistantService
{
    public Task<string> AskAsync(string? prompt, string? systemPrompt, CancellationToken cancellationToken)
    {
        if (!llmClient.IsEnabled)
        {
            throw new ValidationException("LLM tool is disabled. Set AGRONODE_MCP_LLM_ENABLED=true.");
        }

        var text = Guard.RequireText(prompt, "prompt");
        var system = string.IsNullOrWhiteSpace(systemPrompt) ? null : systemPrompt.Trim();

        return llmClient.AskAsync(text, system, cancellationToken);
    }
}
