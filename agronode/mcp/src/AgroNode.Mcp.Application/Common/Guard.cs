namespace AgroNode.Mcp.Application.Common;

internal static class Guard
{
    public static string RequireText(string? value, string field)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            throw new ValidationException($"Missing required argument: {field}");
        }

        return value.Trim();
    }
}
