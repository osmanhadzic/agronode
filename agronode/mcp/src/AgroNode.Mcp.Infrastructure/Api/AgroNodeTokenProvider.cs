using System.Net.Http.Json;
using System.Text.Json;

namespace AgroNode.Mcp.Infrastructure.Api;

public interface IAgroNodeTokenProvider
{
    Task<string?> GetTokenAsync(CancellationToken cancellationToken);

    void Invalidate();
}

/// <summary>
/// Supplies the backend session token: a configured static token, or one obtained from the
/// existing <c>POST /api/auth/login</c> endpoint and cached until shortly before it expires.
/// </summary>
public sealed class AgroNodeTokenProvider(
    IHttpClientFactory httpClientFactory,
    Configuration.AgroNodeOptions options,
    TimeProvider timeProvider) : IAgroNodeTokenProvider
{
    public const string AuthClientName = "agronode-auth";

    private static readonly TimeSpan RefreshMargin = TimeSpan.FromSeconds(60);

    private readonly SemaphoreSlim gate = new(1, 1);
    private string? cachedToken;
    private DateTimeOffset cachedUntil;

    public async Task<string?> GetTokenAsync(CancellationToken cancellationToken)
    {
        if (!string.IsNullOrWhiteSpace(options.ApiToken))
        {
            return options.ApiToken;
        }

        if (string.IsNullOrWhiteSpace(options.LoginEmail) || string.IsNullOrWhiteSpace(options.LoginPassword))
        {
            return null;
        }

        await gate.WaitAsync(cancellationToken);
        try
        {
            if (cachedToken is not null && timeProvider.GetUtcNow() < cachedUntil)
            {
                return cachedToken;
            }

            var client = httpClientFactory.CreateClient(AuthClientName);
            using var response = await client.PostAsJsonAsync(
                "/api/auth/login",
                new { email = options.LoginEmail, password = options.LoginPassword },
                cancellationToken);

            if (!response.IsSuccessStatusCode)
            {
                throw new InvalidOperationException(
                    $"AgroNode backend login failed with status {(int)response.StatusCode}.");
            }

            using var document = JsonDocument.Parse(await response.Content.ReadAsStringAsync(cancellationToken));
            var root = document.RootElement;
            var token = root.TryGetProperty("token", out var tokenElement) ? tokenElement.GetString() : null;

            if (string.IsNullOrWhiteSpace(token))
            {
                throw new InvalidOperationException("AgroNode backend login returned no token.");
            }

            var expiresAt = root.TryGetProperty("expiresAt", out var expiresElement) && expiresElement.TryGetInt64(out var unix)
                ? DateTimeOffset.FromUnixTimeSeconds(unix)
                : timeProvider.GetUtcNow().AddMinutes(5);

            cachedToken = token;
            cachedUntil = expiresAt - RefreshMargin;
            return token;
        }
        finally
        {
            gate.Release();
        }
    }

    public void Invalidate()
    {
        cachedToken = null;
        cachedUntil = default;
    }
}
