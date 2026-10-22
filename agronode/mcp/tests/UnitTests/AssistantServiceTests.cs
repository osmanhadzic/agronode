using AgroNode.Mcp.Application.Assistant;
using AgroNode.Mcp.Application.Common;
using AgroNode.Mcp.UnitTests.Fakes;

namespace AgroNode.Mcp.UnitTests;

public class AssistantServiceTests
{
    [Fact]
    public async Task Ask_WhenDisabled_Throws()
    {
        var service = new AssistantService(new FakeLlmClient(enabled: false));

        await Assert.ThrowsAsync<ValidationException>(() => service.AskAsync("hi", null, CancellationToken.None));
    }

    [Fact]
    public async Task Ask_RequiresPrompt()
    {
        var service = new AssistantService(new FakeLlmClient(enabled: true));

        await Assert.ThrowsAsync<ValidationException>(() => service.AskAsync("  ", null, CancellationToken.None));
    }

    [Fact]
    public async Task Ask_ForwardsTrimmedValues()
    {
        var llm = new FakeLlmClient(enabled: true);
        var service = new AssistantService(llm);

        var answer = await service.AskAsync(" status? ", " be brief ", CancellationToken.None);

        Assert.Equal("answer", answer);
        Assert.Equal("status?", llm.Prompt);
        Assert.Equal("be brief", llm.SystemPrompt);
    }
}
