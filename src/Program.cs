

using Microsoft.Extensions.Configuration;

void Bi
IConfigurationBuilder builder = new ConfigurationBuilder().AddJsonFile("appsettings.json", false, true);
IConfigurationRoot root = builder.Build();


namespace Parltube.Crawler
{
    public class Program
    {
        
    }
}