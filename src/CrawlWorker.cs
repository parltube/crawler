using System.Net;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Logging;

namespace Parltube.Crawler; 

public class CrawlWorker(CrawlerDbContext dbContext, ILogger<CrawlWorker> logger, IConfiguration configuration) {
    // runs the worker  
    public void Run()
    {
        var handler = new SocketsHttpHandler
        {
            MaxConnectionsPerServer = 512, // подбирается эмпирически
            PooledConnectionLifetime = TimeSpan.FromMinutes(5), // обязательно, иначе DNS залипнет
            PooledConnectionIdleTimeout = TimeSpan.FromMinutes(2),
            EnableMultipleHttp2Connections = true, // критично, если сайт на HTTP/2
            ConnectTimeout = TimeSpan.FromSeconds(5),
            AutomaticDecompression = DecompressionMethods.All,
            UseCookies = false, // CookieContainer — глобальный лок, убивает throughput
            UseProxy = false, // если прокси нет на этом инстансе
            ActivityHeadersPropagator = null, // убирает лишний overhead трейсинга
        };
        var httpclient = new HttpClient(handler); 
    }
}