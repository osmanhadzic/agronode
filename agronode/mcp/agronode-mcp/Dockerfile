FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

COPY src/AgroNode.McpServer/AgroNode.McpServer.csproj src/AgroNode.McpServer/
RUN dotnet restore src/AgroNode.McpServer/AgroNode.McpServer.csproj

COPY src/AgroNode.McpServer/ src/AgroNode.McpServer/
RUN dotnet publish src/AgroNode.McpServer/AgroNode.McpServer.csproj -c Release -o /app/publish /p:UseAppHost=false

FROM mcr.microsoft.com/dotnet/runtime:8.0 AS runtime
WORKDIR /app

COPY --from=build /app/publish .

ENTRYPOINT ["dotnet", "AgroNode.McpServer.dll"]
