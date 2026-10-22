namespace AgroNode.Mcp.Application.Common;

public sealed class ValidationException(string message) : Exception(message);
