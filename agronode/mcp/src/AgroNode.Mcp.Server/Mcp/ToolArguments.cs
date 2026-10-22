using System.Text.Json.Nodes;
using AgroNode.Mcp.Application.Common;

namespace AgroNode.Mcp.Server.Mcp;

internal static class ToolArguments
{
    public static string? String(JsonObject? source, string field)
    {
        var node = source?[field];
        if (node is null)
        {
            return null;
        }

        if (node is JsonValue value && value.TryGetValue<string>(out var text))
        {
            return text;
        }

        throw new ValidationException($"Argument '{field}' must be a string.");
    }

    public static int? Int(JsonObject? source, string field)
    {
        var node = source?[field];
        if (node is null)
        {
            return null;
        }

        if (node is JsonValue value && value.TryGetValue<int>(out var number))
        {
            return number;
        }

        throw new ValidationException($"Argument '{field}' must be an integer.");
    }

    public static JsonObject Schema(JsonObject properties, params string[] required)
    {
        var schema = new JsonObject
        {
            ["type"] = "object",
            ["properties"] = properties
        };

        if (required.Length > 0)
        {
            var array = new JsonArray();
            foreach (var name in required)
            {
                array.Add(name);
            }

            schema["required"] = array;
        }

        schema["additionalProperties"] = false;
        return schema;
    }

    public static JsonObject Property(string type, string? description = null, int? minLength = null, int? minimum = null, int? maximum = null)
    {
        var property = new JsonObject { ["type"] = type };

        if (description is not null) property["description"] = description;
        if (minLength is not null) property["minLength"] = minLength;
        if (minimum is not null) property["minimum"] = minimum;
        if (maximum is not null) property["maximum"] = maximum;

        return property;
    }
}
