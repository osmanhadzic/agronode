# AgroNode MCP Server (.NET 8)

stdio MCP server (`Content-Length` framing, protocol 2024-11-05) that calls the existing Go backend REST API.

## Layout

```
AgroNode.Mcp.sln
src/
  AgroNode.Mcp.Server          MCP host, tools (Tools/), transport (Mcp/), security (Security/)
  AgroNode.Mcp.Application     services + gateway abstractions (Devices, Telemetry, Assistant)
  AgroNode.Mcp.Infrastructure  backend API client, auth, LLM clients, configuration, DI
tests/
  UnitTests                    Application services (fakes)
  IntegrationTests             API client (stub HTTP backend), stdio server, config
```

References: Server -> Application, Infrastructure; Infrastructure -> Application.

## Tools

`get_devices`, `get_device_status`, `get_latest_telemetry`, `get_telemetry_history` (read-only), `ask_llm` (optional, `AGRONODE_MCP_LLM_ENABLED=true`).
Command tools are blocked unless `AGRONODE_MCP_ALLOW_COMMANDS=true`; none exist yet.

## Configuration (env vars override `appsettings.json`)

| Variable | Purpose |
|---|---|
| `AGRONODE_MCP_REST_BASE_URL` | Backend URL (default `http://localhost:8080`) |
| `AGRONODE_MCP_ORGANIZATION_ID` | Sent as `X-Organization-ID` |
| `AGRONODE_MCP_API_TOKEN` | Static bearer token, or |
| `AGRONODE_MCP_LOGIN_EMAIL` / `AGRONODE_MCP_LOGIN_PASSWORD` | Login via `POST /api/auth/login` |
| `AGRONODE_MCP_ALLOW_COMMANDS` | Enable command tools (default false) |

## Run / test

```bash
dotnet run --project src/AgroNode.Mcp.Server
dotnet test AgroNode.Mcp.sln
docker compose up -d --build   # from repo root; includes agronode-mcp
```

## Backend gaps

- No irrigation-activation or alert endpoints exist, so `IrrigationTools`/`AlertTools` are not implemented.
- Backend telemetry has no `limit` and sensor endpoint has no date filter; both applied client-side.
