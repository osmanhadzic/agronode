namespace AgroNode.McpServer.Clients;

internal interface ILlmClient
{
    bool IsEnabled { get; }
    Task<string> AskAsync(string prompt, string? systemPromptOverride, CancellationToken cancellationToken);
}
