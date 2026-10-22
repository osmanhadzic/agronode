using System.Text;
using System.Text.Json.Nodes;
using AgroNode.Mcp.Server.Mcp;
using Microsoft.Extensions.DependencyInjection;

namespace AgroNode.Mcp.IntegrationTests;

public class McpStdioServerTests
{
    [Fact]
    public async Task ToolsList_ExposesReadOnlyToolsOnly()
    {
        var responses = await RunAsync(new StubBackendHandler(_ => StubBackendHandler.Json("[]")),
            Request(1, "tools/list"));

        var names = responses[0]["result"]!["tools"]!.AsArray().Select(tool => tool!["name"]!.GetValue<string>()).ToArray();

        Assert.Equal(
            ["get_devices", "get_device_status", "get_latest_telemetry", "get_telemetry_history", "ask_llm"],
            names);
    }

    [Fact]
    public async Task Initialize_ReturnsServerInfo()
    {
        var responses = await RunAsync(new StubBackendHandler(_ => StubBackendHandler.Json("[]")),
            Request(1, "initialize"));

        Assert.Equal("agronode-mcp-dotnet", responses[0]["result"]!["serverInfo"]!["name"]!.GetValue<string>());
        Assert.Equal("2024-11-05", responses[0]["result"]!["protocolVersion"]!.GetValue<string>());
    }

    [Fact]
    public async Task ToolsCall_GetDeviceStatus_ReturnsBackendJson()
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json("{\"deviceId\":\"d1\",\"status\":\"online\"}"));

        var responses = await RunAsync(backend,
            Request(1, "tools/call", new JsonObject
            {
                ["name"] = "get_device_status",
                ["arguments"] = new JsonObject { ["device_id"] = "d1" }
            }));

        var result = responses[0]["result"]!;
        Assert.False(result["isError"]!.GetValue<bool>());
        Assert.Contains("online", result["content"]![0]!["text"]!.GetValue<string>());
        Assert.Equal("/api/devices/d1", backend.Requests.Single().RequestUri!.AbsolutePath);
    }

    [Fact]
    public async Task ToolsCall_InvalidArguments_ReturnToolErrorWithoutCallingBackend()
    {
        var backend = new StubBackendHandler(_ => StubBackendHandler.Json("{}"));

        var responses = await RunAsync(backend,
            Request(1, "tools/call", new JsonObject { ["name"] = "get_device_status", ["arguments"] = new JsonObject() }),
            Request(2, "tools/call", new JsonObject { ["name"] = "does_not_exist" }),
            Request(3, "tools/call", new JsonObject { ["name"] = "ask_llm", ["arguments"] = new JsonObject { ["prompt"] = "hi" } }));

        Assert.All(responses, response => Assert.True(response["result"]!["isError"]!.GetValue<bool>()));
        Assert.Contains("device_id", responses[0]["result"]!["content"]![0]!["text"]!.GetValue<string>());
        Assert.Contains("Unknown tool", responses[1]["result"]!["content"]![0]!["text"]!.GetValue<string>());
        Assert.Contains("disabled", responses[2]["result"]!["content"]![0]!["text"]!.GetValue<string>());
        Assert.Empty(backend.Requests);
    }

    [Fact]
    public async Task UnknownMethod_ReturnsJsonRpcError()
    {
        var responses = await RunAsync(new StubBackendHandler(_ => StubBackendHandler.Json("{}")), Request(1, "nope"));

        Assert.Equal(-32601, responses[0]["error"]!["code"]!.GetValue<int>());
    }

    private static string Request(int id, string method, JsonObject? parameters = null)
    {
        var request = new JsonObject { ["jsonrpc"] = "2.0", ["id"] = id, ["method"] = method };
        if (parameters is not null)
        {
            request["params"] = parameters;
        }

        return request.ToJsonString();
    }

    private static async Task<List<JsonNode>> RunAsync(StubBackendHandler backend, params string[] requests)
    {
        await using var provider = TestHost.Build(backend);
        var server = provider.GetRequiredService<McpStdioServer>();

        var input = new MemoryStream();
        foreach (var request in requests)
        {
            var body = Encoding.UTF8.GetBytes(request);
            input.Write(Encoding.ASCII.GetBytes($"Content-Length: {body.Length}\r\n\r\n"));
            input.Write(body);
        }

        input.Position = 0;
        var output = new MemoryStream();
        await server.RunAsync(input, output, CancellationToken.None);

        return ParseFrames(output.ToArray());
    }

    private static List<JsonNode> ParseFrames(byte[] data)
    {
        var text = Encoding.UTF8.GetString(data);
        var frames = new List<JsonNode>();
        var position = 0;

        while (position < text.Length)
        {
            var headerEnd = text.IndexOf("\r\n\r\n", position, StringComparison.Ordinal);
            var length = int.Parse(text[(position + "Content-Length: ".Length)..headerEnd]);
            var bodyStart = headerEnd + 4;
            var bodyBytes = Encoding.UTF8.GetBytes(text[bodyStart..]);
            frames.Add(JsonNode.Parse(Encoding.UTF8.GetString(bodyBytes, 0, length))!);
            position = bodyStart + Encoding.UTF8.GetString(bodyBytes, 0, length).Length;
        }

        return frames;
    }
}
