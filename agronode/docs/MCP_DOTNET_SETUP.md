# .NET MCP Setup Guide

This guide describes how to bootstrap and run an AgroNode MCP server in .NET, with MCP tools calling AgroNode backend APIs/gRPC services.

---

## 1) Prerequisites

- .NET SDK 8.0+
- Docker + Docker Compose
- Running AgroNode stack (`backend`, `postgres`, `mosquitto`)
- Basic access token/session strategy already used by AgroNode clients

Check your SDK:

```bash
dotnet --version
```

Start backend dependencies:

```bash
cd agronode
docker compose up -d --build
```

---

## 2) Create MCP server project structure

From repository root:

```bash
cd agronode
mkdir -p mcp/agronode-mcp/src
cd mcp/agronode-mcp/src
dotnet new web -n AgroNode.McpServer
```

Recommended layout:

```text
agronode/
└── mcp/
    └── agronode-mcp/
        ├── src/
        │   └── AgroNode.McpServer/
        │       ├── Program.cs
        │       ├── appsettings.json
        │       ├── Tools/
        │       └── Clients/
        └── README.md
```

---

## 3) Configure backend endpoints

In `appsettings.json`, define AgroNode backend endpoints used by MCP tools:

```json
{
  "AgroNode": {
    "RestBaseUrl": "http://localhost:8080",
    "GrpcBaseUrl": "http://localhost:50051",
    "OrganizationId": "default-org"
  },
  "Mcp": {
    "Transport": "stdio"
  }
}
```

Notes:
- Keep environment-specific values in `appsettings.Development.json` or env vars.
- Do not hardcode secrets.

---

## 4) Implement MCP tools (minimum set)

Create tools that map to existing AgroNode capabilities:

- `get_devices()`
- `get_device_status(device_id)`
- `get_latest_telemetry(device_id)`
- `get_telemetry_history(device_id, from, to)`
- `get_automation_rules()`
- `get_rule_executions(device_id)`

Implementation rules:

- Enforce org scope (`X-Organization-ID`) on every backend call.
- Validate input (`device_id`, date ranges, limits) before sending upstream.
- Return structured, stable JSON responses for each tool.
- Convert backend errors to MCP-friendly error messages without leaking internals.

---

## 5) Security baseline

Required controls:

- Never expose DB/MQTT credentials via tool responses.
- Never expose private keys or certificate secrets.
- Add allowlist for callable backend routes.
- Add request timeout and retry policy (bounded, no infinite retry).
- Log tool invocation metadata (tool name, duration, status) without sensitive payload dumps.

---

## 6) Local run

Run MCP server:

```bash
cd agronode/mcp/agronode-mcp/src/AgroNode.McpServer
dotnet run
```

If the MCP host/client expects stdio transport, run it as a child process from the host configuration.

---

## 7) Validation checklist

- MCP server process starts without configuration errors.
- `get_devices()` returns data from AgroNode backend.
- Invalid `device_id` returns clear validation error.
- Backend outage returns graceful MCP error (no crash).
- Organization scoping is applied on every request.

---

## 8) Suggested next docs

After implementation, add:

- `mcp/agronode-mcp/README.md` (developer runbook)
- Tool contract reference (input/output schema per tool)
- Incident troubleshooting (`timeout`, `auth`, `backend unavailable`)
