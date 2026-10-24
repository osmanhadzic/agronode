using System.Net;
using System.Text;

namespace AgroNode.Mcp.IntegrationTests;

internal sealed class StubBackendHandler : HttpMessageHandler
{
    private readonly Func<HttpRequestMessage, HttpResponseMessage> responder;

    public List<HttpRequestMessage> Requests { get; } = [];

    public StubBackendHandler(Func<HttpRequestMessage, HttpResponseMessage> responder) => this.responder = responder;

    public static HttpResponseMessage Json(string body, HttpStatusCode status = HttpStatusCode.OK) =>
        new(status) { Content = new StringContent(body, Encoding.UTF8, "application/json") };

    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        Requests.Add(request);
        return Task.FromResult(responder(request));
    }
}
