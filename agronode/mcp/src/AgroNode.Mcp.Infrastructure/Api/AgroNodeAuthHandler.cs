using System.Net;
using System.Net.Http.Headers;

namespace AgroNode.Mcp.Infrastructure.Api;

internal sealed class AgroNodeAuthHandler(IAgroNodeTokenProvider tokenProvider) : DelegatingHandler
{
    protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        var token = await tokenProvider.GetTokenAsync(cancellationToken);
        if (!string.IsNullOrWhiteSpace(token))
        {
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);
        }

        var response = await base.SendAsync(request, cancellationToken);

        if (response.StatusCode == HttpStatusCode.Unauthorized)
        {
            tokenProvider.Invalidate();
        }

        return response;
    }
}
