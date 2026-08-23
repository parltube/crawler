using System.Net;
using System.Security.Authentication.ExtendedProtection;
using System.Security.Cryptography;
using System.Threading.Channels;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Logging;

namespace Parltube.Crawler; 

public class CrawlWorker(CrawlerDbContext dbContext, ILogger<CrawlWorker> logger, IConfiguration configuration) {
    // runs the worker  
    public List<HttpClient> HttpClients = []; 
    
    public async Task Run()
    {
        var proxies = []; 

        foreach (var proxy in proxies)
        {
            HttpClients.Add(new HttpClient(new SocketsHttpHandler() {
                MaxConnectionsPerServer = 512, // подбирается эмпирически
                PooledConnectionLifetime = TimeSpan.FromMinutes(5), // обязательно, иначе DNS залипнет
                PooledConnectionIdleTimeout = TimeSpan.FromMinutes(2),
                EnableMultipleHttp2Connections = true, // критично, если сайт на HTTP/2
                ConnectTimeout = TimeSpan.FromSeconds(5),
                AutomaticDecompression = DecompressionMethods.All,
                UseCookies = false, // CookieContainer — глобальный лок, убивает throughput
                UseProxy = true, // если прокси нет на этом инстансе
                ActivityHeadersPropagator = null, // убирает лишний overhead трейсинга, 
                Proxy = new WebProxy(BypassOnLocal: true, ), // беру из списка и все
            }));
        }

        var index = 0;
        Channel<CrawlRequest> channel = Channel.CreateUnbounded<CrawlRequest>(new UnboundedChannelOptions()
        {
            SingleReader = false,
            SingleWriter = false,
            AllowSynchronousContinuations = false,
        }); 
        
        Task producer = ProduceAsync(channel.Writer);
        Task consumer = ConsumeAsync(channel.Reader);
        
        foreach (var video in itervideos)
        {
            var nextItem = httpClients[Interlocked.Increment(ref index) % clientCount];
            nextItem.PostAsync(); 
        }
    }

    public async Task ProduceAsync(ChannelWriter<CrawlRequest> channel)
    {
        // finds anything related and needed that WASN'T indexed nor properly saved  
        while (true)
        {
            
        }
        
        channel.Complete();
    }

    public async Task ConsumeAsync(ChannelReader<CrawlRequest> reader)
    {
        // REQUESTS from youtube all metadata saves it to db 
        await foreach (CrawlRequest item in reader.ReadAllAsync())
        {
            
        }
    }
}

public readonly struct CrawlRequest(string VideoId, string ChannelId);

public readonly struct SubtitleFrame(TimeOnly TimeStamp, string Text);  