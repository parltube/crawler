package crawler

import tls_client "github.com/bogdanfinn/tls-client"

// consistent worker struct that helps with thread pool api
type Worker struct {
	httpClient *tls_client.HttpClient
}

func (*Worker) Run(context *CrawlerContext) {

}
