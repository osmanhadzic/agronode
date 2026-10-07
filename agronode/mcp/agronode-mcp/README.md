# AgroNode .NET MCP Server

Minimal MCP server implementation in .NET (stdio transport), designed to expose read-only AgroNode tools to an MCP host.

---

## Project Structure

```text
mcp/agronode-mcp/
└── src/
    └── AgroNode.McpServer/
        ├── AgroNode.McpServer.csproj
        ├── Program.cs
        ├── appsettings.json
        ├── Clients/
        └── Options/
```

---

## Implemented Tools

- `get_devices`
- `get_device_status`
- `get_latest_telemetry`
- `get_telemetry_history`

All requests are organization-scoped with `X-Organization-ID`.

---

## Configuration

Environment variables (override `appsettings.json`):

- `AGRONODE_MCP_REST_BASE_URL` (default: `http://localhost:8080`)
- `AGRONODE_MCP_ORGANIZATION_ID` (default: `1`)
- `AGRONODE_MCP_TIMEOUT_SECONDS` (default: `15`)
- `AGRONODE_MCP_SERVER_NAME` (default: `agronode-mcp-dotnet`)
- `AGRONODE_MCP_SERVER_VERSION` (default: `0.1.0`)
- `AGRONODE_MCP_PROTOCOL_VERSION` (default: `2024-11-05`)

---

## Run Locally

From repository root:

```bash
cd mcp/agronode-mcp/src/AgroNode.McpServer
dotnet restore
dotnet run
```

The process communicates over stdio using MCP JSON-RPC framing.

---

## Example MCP Host Configuration

Use this command as MCP server command in your host:

```bash
dotnet run --project /absolute/path/to/agronode/mcp/agronode-mcp/src/AgroNode.McpServer
```

If backend runs in Docker and MCP host runs on Linux host machine, keep:

- `AGRONODE_MCP_REST_BASE_URL=http://localhost:8080`

If MCP server runs in container, use internal service DNS:

- `AGRONODE_MCP_REST_BASE_URL=http://backend:8080`
