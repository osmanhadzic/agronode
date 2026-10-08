using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using AgroNode.McpServer.Clients;
using AgroNode.McpServer.Options;

namespace AgroNode.McpServer;

internal sealed class McpStdioServer(IAgroNodeApiClient client, McpOptions options)
{
    public async Task RunAsync(Stream input, Stream output, CancellationToken cancellationToken)
    {
        while (!cancellationToken.IsCancellationRequested && await TryReadFrameAsync(input, cancellationToken) is { } payload)
        {
            var response = await HandleRequestAsync(payload, cancellationToken);
            if (response is null)
            {
                continue;
            }

            await WriteFrameAsync(output, response, cancellationToken);
        }
    }

    private async Task<string?> TryReadFrameAsync(Stream input, CancellationToken cancellationToken)
    {
        var headers = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);

        while (true)
        {
            var line = await ReadLineAsync(input, cancellationToken);
            if (line is null)
            {
                return null;
            }

            if (line.Length == 0)
            {
                break;
            }

            var separatorIndex = line.IndexOf(':');
            if (separatorIndex <= 0)
            {
                continue;
            }

            var name = line[..separatorIndex].Trim();
            var value = line[(separatorIndex + 1)..].Trim();
            headers[name] = value;
        }

        if (!headers.TryGetValue("Content-Length", out var contentLengthValue) ||
            !int.TryParse(contentLengthValue, out var contentLength) ||
            contentLength < 0)
        {
            return null;
        }

        var payloadBuffer = new byte[contentLength];
        var bytesRead = 0;

        while (bytesRead < contentLength)
        {
            var chunk = await input.ReadAsync(payloadBuffer.AsMemory(bytesRead, contentLength - bytesRead), cancellationToken);
            if (chunk == 0)
            {
                return null;
            }

            bytesRead += chunk;
        }

        return Encoding.UTF8.GetString(payloadBuffer);
    }

    private static async Task<string?> ReadLineAsync(Stream input, CancellationToken cancellationToken)
    {
        var bytes = new List<byte>();

        while (true)
        {
            var buffer = new byte[1];
            var read = await input.ReadAsync(buffer.AsMemory(0, 1), cancellationToken);

            if (read == 0)
            {
                return bytes.Count == 0 ? null : Encoding.ASCII.GetString(bytes.ToArray());
            }

            var current = buffer[0];
            if (current == '\n')
            {
                break;
            }

            if (current != '\r')
            {
                bytes.Add(current);
            }
        }

        return Encoding.ASCII.GetString(bytes.ToArray());
    }

    private static async Task WriteFrameAsync(Stream output, string payload, CancellationToken cancellationToken)
    {
        var bodyBytes = Encoding.UTF8.GetBytes(payload);
        var header = $"Content-Length: {bodyBytes.Length}\r\n\r\n";
        var headerBytes = Encoding.ASCII.GetBytes(header);

        await output.WriteAsync(headerBytes.AsMemory(0, headerBytes.Length), cancellationToken);
        await output.WriteAsync(bodyBytes.AsMemory(0, bodyBytes.Length), cancellationToken);
        await output.FlushAsync(cancellationToken);
    }

    private async Task<string?> HandleRequestAsync(string payload, CancellationToken cancellationToken)
    {
        JsonNode? root;

        try
        {
            root = JsonNode.Parse(payload);
        }
        catch
        {
            return JsonSerializer.Serialize(new JsonObject
            {
                ["jsonrpc"] = "2.0",
                ["id"] = null,
                ["error"] = new JsonObject
                {
                    ["code"] = -32700,
                    ["message"] = "Parse error"
                }
            });
        }

        var method = root?["method"]?.GetValue<string>();
        var id = root?["id"];

        if (string.IsNullOrWhiteSpace(method))
        {
            return BuildError(id, -32600, "Invalid request");
        }

        if (id is null)
        {
            return null;
        }

        var parameters = root?["params"] as JsonObject;

        try
        {
            return method switch
            {
                "initialize" => BuildResult(id, BuildInitializeResult()),
                "tools/list" => BuildResult(id, new JsonObject { ["tools"] = BuildToolsDefinition() }),
                "tools/call" => BuildResult(id, await HandleToolCallAsync(parameters, cancellationToken)),
                _ => BuildError(id, -32601, "Method not found")
            };
        }
        catch (Exception ex)
        {
            return BuildResult(id, new JsonObject
            {
                ["content"] = new JsonArray
                {
                    new JsonObject
                    {
                        ["type"] = "text",
                        ["text"] = ex.Message
                    }
                },
                ["isError"] = true
            });
        }
    }

    private JsonObject BuildInitializeResult() => new()
    {
        ["protocolVersion"] = options.ProtocolVersion,
        ["capabilities"] = new JsonObject
        {
            ["tools"] = new JsonObject
            {
                ["listChanged"] = false
            }
        },
        ["serverInfo"] = new JsonObject
        {
            ["name"] = options.ServerName,
            ["version"] = options.ServerVersion
        }
    };

    private static JsonArray BuildToolsDefinition() =>
    [
        new JsonObject
        {
            ["name"] = "get_devices",
            ["description"] = "Returns AgroNode devices from backend API.",
            ["inputSchema"] = new JsonObject
            {
                ["type"] = "object",
                ["properties"] = new JsonObject(),
                ["additionalProperties"] = false
            }
        },
        new JsonObject
        {
            ["name"] = "get_device_status",
            ["description"] = "Returns detailed status for a single device.",
            ["inputSchema"] = new JsonObject
            {
                ["type"] = "object",
                ["properties"] = new JsonObject
                {
                    ["device_id"] = new JsonObject
                    {
                        ["type"] = "string",
                        ["minLength"] = 1
                    }
                },
                ["required"] = new JsonArray("device_id"),
                ["additionalProperties"] = false
            }
        },
        new JsonObject
        {
            ["name"] = "get_latest_telemetry",
            ["description"] = "Returns latest telemetry snapshot for a device.",
            ["inputSchema"] = new JsonObject
            {
                ["type"] = "object",
                ["properties"] = new JsonObject
                {
                    ["device_id"] = new JsonObject
                    {
                        ["type"] = "string",
                        ["minLength"] = 1
                    }
                },
                ["required"] = new JsonArray("device_id"),
                ["additionalProperties"] = false
            }
        },
        new JsonObject
        {
            ["name"] = "get_telemetry_history",
            ["description"] = "Returns telemetry history for a device in a time range.",
            ["inputSchema"] = new JsonObject
            {
                ["type"] = "object",
                ["properties"] = new JsonObject
                {
                    ["device_id"] = new JsonObject
                    {
                        ["type"] = "string",
                        ["minLength"] = 1
                    },
                    ["from"] = new JsonObject
                    {
                        ["type"] = "string",
                        ["description"] = "ISO-8601 timestamp"
                    },
                    ["to"] = new JsonObject
                    {
                        ["type"] = "string",
                        ["description"] = "ISO-8601 timestamp"
                    },
                    ["sensor_id"] = new JsonObject
                    {
                        ["type"] = "string"
                    },
                    ["limit"] = new JsonObject
                    {
                        ["type"] = "integer",
                        ["minimum"] = 1,
                        ["maximum"] = 5000
                    }
                },
                ["required"] = new JsonArray("device_id", "from", "to"),
                ["additionalProperties"] = false
            }
        }
    ];

    private async Task<JsonObject> HandleToolCallAsync(JsonObject? parameters, CancellationToken cancellationToken)
    {
        var name = parameters?["name"]?.GetValue<string>();
        var arguments = parameters?["arguments"] as JsonObject;

        if (string.IsNullOrWhiteSpace(name))
        {
            throw new InvalidOperationException("Tool call is missing tool name.");
        }

        var result = name switch
        {
            "get_devices" => await client.GetDevicesAsync(cancellationToken),
            "get_device_status" => await client.GetDeviceStatusAsync(
                RequireString(arguments, "device_id"),
                cancellationToken),
            "get_latest_telemetry" => await client.GetLatestTelemetryAsync(
                RequireString(arguments, "device_id"),
                cancellationToken),
            "get_telemetry_history" => await client.GetTelemetryHistoryAsync(
                RequireString(arguments, "device_id"),
                RequireIsoTimestamp(arguments, "from"),
                RequireIsoTimestamp(arguments, "to"),
                OptionalString(arguments, "sensor_id"),
                OptionalInt(arguments, "limit"),
                cancellationToken),
            _ => throw new InvalidOperationException($"Unknown tool '{name}'.")
        };

        return new JsonObject
        {
            ["content"] = new JsonArray
            {
                new JsonObject
                {
                    ["type"] = "text",
                    ["text"] = result
                }
            },
            ["isError"] = false
        };
    }

    private static string RequireString(JsonObject? source, string field)
    {
        var value = source?[field]?.GetValue<string>();
        if (string.IsNullOrWhiteSpace(value))
        {
            throw new InvalidOperationException($"Missing required argument: {field}");
        }

        return value;
    }

    private static string? OptionalString(JsonObject? source, string field)
    {
        var value = source?[field]?.GetValue<string>();
        return string.IsNullOrWhiteSpace(value) ? null : value;
    }

    private static int? OptionalInt(JsonObject? source, string field)
    {
        var valueNode = source?[field];
        if (valueNode is null)
        {
            return null;
        }

        return valueNode.GetValue<int>();
    }

    private static string RequireIsoTimestamp(JsonObject? source, string field)
    {
        var value = RequireString(source, field);
        if (!DateTimeOffset.TryParse(value, out _))
        {
            throw new InvalidOperationException($"Argument '{field}' must be a valid ISO-8601 timestamp.");
        }

        return value;
    }

    private static string BuildResult(JsonNode? id, JsonObject result) =>
        JsonSerializer.Serialize(new JsonObject
        {
            ["jsonrpc"] = "2.0",
            ["id"] = id?.DeepClone(),
            ["result"] = result
        });

    private static string BuildError(JsonNode? id, int code, string message) =>
        JsonSerializer.Serialize(new JsonObject
        {
            ["jsonrpc"] = "2.0",
            ["id"] = id?.DeepClone(),
            ["error"] = new JsonObject
            {
                ["code"] = code,
                ["message"] = message
            }
        });
}
