package proxy

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/crispyLL/parltube-crawler/internal/config"
)

type RevolvingProxy struct {
	httpClients []tls_client.HttpClient

	lock sync.Mutex
}

func NewRevolvingProxy(cfg *config.Config) (IRevolvingProxy, error) {
	proxy := &RevolvingProxy{httpClients: []tls_client.HttpClient{}}

	if cfg.ProxyCount <= 0 {
		return nil, fmt.Errorf("revolvingproxy: proxy count is 0")
	}

	var httpClientOptions []tls_client.HttpClientOption

	proxyPort := 1080

	for i := 0; i < cfg.ProxyCount; i++ {
		httpClientOptions = []tls_client.HttpClientOption{
			tls_client.WithClientProfile(profiles.Chrome_152),
			tls_client.WithProxyUrl("socks5://vpn:" + (strconv.Itoa(proxyPort + i))),
			tls_client.WithTimeoutSeconds(20),
			// real chrome shuffles tls extensions on every handshake since v110
			tls_client.WithRandomTLSExtensionOrder(),
			// separate jar per proxy so cookies of different exit ips never mix
			tls_client.WithCookieJar(tls_client.NewCookieJar()),
			// used only when request has no headers set
			tls_client.WithDefaultHeaders(chromeHeaders()),
		}

		httpClient, err := tls_client.NewHttpClient(tls_client.NewLogger(), httpClientOptions...)

		if err != nil {
			return nil, err
		}

		proxy.httpClients = append(proxy.httpClients, httpClient)
	}

	return proxy, nil
}

// headers of chrome 152 on windows doing top level navigation, must match profiles.Chrome_152
func chromeHeaders() fhttp.Header {
	return fhttp.Header{
		"sec-ch-ua":                 {`"Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {"en-US,en;q=0.9"},
		"priority":                  {"u=0, i"},
		fhttp.HeaderOrderKey: {
			"sec-ch-ua",
			"sec-ch-ua-mobile",
			"sec-ch-ua-platform",
			"upgrade-insecure-requests",
			"user-agent",
			"accept",
			"sec-fetch-site",
			"sec-fetch-mode",
			"sec-fetch-user",
			"sec-fetch-dest",
			"accept-encoding",
			"accept-language",
			"cookie",
			"priority",
		},
	}
}

func (r *RevolvingProxy) Next() (tls_client.HttpClient, int) {
	if len(r.httpClients) == 0 {
		panic("something gone very very wrong, no httpclients")
	}
	index := rand.Int() % len(r.httpClients)
	return r.httpClients[index], index
}

func (r *RevolvingProxy) Change(index int) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, err := http.Get("http://vpn:8080/cgi-bin/rotate?" + strconv.Itoa(index))
	if err != nil {
		return err
	}

	return nil
}
