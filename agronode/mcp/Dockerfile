FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

COPY src/AgroNode.Mcp.Application/AgroNode.Mcp.Application.csproj src/AgroNode.Mcp.Application/
COPY src/AgroNode.Mcp.Infrastructure/AgroNode.Mcp.Infrastructure.csproj src/AgroNode.Mcp.Infrastructure/
COPY src/AgroNode.Mcp.Server/AgroNode.Mcp.Server.csproj src/AgroNode.Mcp.Server/
RUN dotnet restore src/AgroNode.Mcp.Server/AgroNode.Mcp.Server.csproj

COPY src/ src/
RUN dotnet publish src/AgroNode.Mcp.Server/AgroNode.Mcp.Server.csproj -c Release -o /app/publish --no-restore /p:UseAppHost=false

FROM mcr.microsoft.com/dotnet/runtime:8.0 AS runtime
WORKDIR /app

COPY --from=build /app/publish .

ENTRYPOINT ["dotnet", "AgroNode.Mcp.Server.dll"]
