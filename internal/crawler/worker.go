package crawler

import (
	"context"

	"github.com/crispyLL/parltube-crawler/internal/config"
	"github.com/crispyLL/parltube-crawler/internal/proxy"
	"github.com/crispyLL/parltube-crawler/internal/structs"
)

// consistent worker struct that helps with thread pool api
type Worker struct {
	RevolvingProxy    proxy.IRevolvingProxy
	TrashChannel      chan any // ладно похуй. мусорный канал и так мусорный. придется вручную добавлять обработчики.
	LoopbackIdParsing <-chan structs.ContentId
	config            *config.Config
}

func NewWorker(proxy proxy.IRevolvingProxy) *Worker {
	return &Worker{RevolvingProxy: proxy}
}

// works this way frontier writes to channel with backpressure.
// workers sends data to root crawler, worker is pretty stupid.
func (w *Worker) Run(context *CrawlerContext, outsideContext context.Context) error {
	httpClient, _ := w.RevolvingProxy.Next()

	nextChannel, err := context.CrawlStorage.GetCursor()

	// it will not stop until signal from outcontext is passed
	for {

	}
}

func (w *Worker) RunFromIds(context *CrawlerContext, outsideContext context.Context) {
	// this will use loopback channel to parse through channels. ONE context at a time
	for {
		var typeId structs.ContentId = <-context.CrawlingIdChannel
		println("typeId:", typeId)
	}
}
