namespace AgroNode.Mcp.Application.Assistant;

public interface ILlmClient
{
    bool IsEnabled { get; }

    Task<string> AskAsync(string prompt, string? systemPromptOverride, CancellationToken cancellationToken);
}
