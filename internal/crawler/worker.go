package crawler

import (
	"context"

	"github.com/crispyLL/parltube-crawler/internal/proxy"
)

// consistent worker struct that helps with thread pool api
type Worker struct {
	RevolvingProxy proxy.IRevolvingProxy
	TrashChannel   chan any // ладно похуй. мусорный канал и так мусорный. придется вручную добавлять обработчики.
}

func NewWorker(proxy proxy.IRevolvingProxy) *Worker {
	return &Worker{RevolvingProxy: proxy}
}

// works this way frontier writes to channel with backpressure.
// workers sends data to root crawler, worker is pretty stupid.
func (w *Worker) Run(context *CrawlerContext, outsideContext context.Context) {
	httpClient, _ := w.RevolvingProxy.Next()

	nextChannel := w.CrawlStorage.Get()

	// it will not stop until signal from outcontext is passed
	for {
		resp, err := httpClient.Get("")
		if err != nil {
			outsideContext.Err()
			return
		}

		resp.Body
	}
}
