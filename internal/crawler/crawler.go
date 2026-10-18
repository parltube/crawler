// Package app wires the crawler components together and runs them.
package crawler

import (
	"context"
	"errors"
	"sync"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/crispyLL/parltube-crawler/internal/config"
	"github.com/crispyLL/parltube-crawler/internal/proxy"
	"github.com/crispyLL/parltube-crawler/internal/storage"
	"github.com/crispyLL/parltube-crawler/internal/structs"
)

// what we give to each worker
type CrawlerContext struct {
	GlobalStopChannel <-chan struct{}
	Proxies           proxy.IRevolvingProxy // struct that revolves Mullvad proxies, abstraction

	// i should really replace this with UoW pattern but i m lazy to refactor
	CommentsStorage *storage.BatchedStorage[structs.CommentsMetadata]
	VideoMetaData   *storage.BatchedStorage[structs.VideoMetaData]
	ChannelMetadata *storage.BatchedStorage[structs.ChannelMetadata]
	PlayerMetadata  *storage.BatchedStorage[structs.PlayerMetadata]
	BrowseMetadata  *storage.BatchedStorage[structs.BrowseMetadata]
	CrawlState      *storage.CrawlState
}

func NewCrawlerContext(proxy proxy.IRevolvingProxy, conn driver.Conn, cfg *config.Config) *CrawlerContext {
	return &CrawlerContext{
		GlobalStopChannel: make(<-chan struct{}, 100),
		CommentsStorage:   storage.NewBatchedStorage[structs.CommentsMetadata](conn, cfg),
		VideoMetaData:     storage.NewBatchedStorage[structs.VideoMetaData](conn, cfg),
		ChannelMetadata:   storage.NewBatchedStorage[structs.ChannelMetadata](conn, cfg),
		PlayerMetadata:    storage.NewBatchedStorage[structs.PlayerMetadata](conn, cfg),
		BrowseMetadata:    storage.NewBatchedStorage[structs.BrowseMetadata](conn, cfg),
	}
}

func (*CrawlerContext) Clear() {

}

// Run starts the crawler and blocks until ctx is cancelled.
type Crawler struct {
	MaxWorkerCount     uint
	CurrentWorkerCount uint
	once               sync.Once
	lock               sync.Mutex
	Context            CrawlerContext
	Workers            []*Worker
}

func NewCrawler(MaxWorkerCount uint, proxy proxy.IRevolvingProxy) (*Crawler, error) {
	workers := make([]*Worker, MaxWorkerCount)

	for i := 1; i < int(MaxWorkerCount); i++ {

		workers = append(workers, NewWorker(proxy))
	}

	return &Crawler{
		MaxWorkerCount: MaxWorkerCount,
		Workers:        make([]*Worker, MaxWorkerCount),
	}, nil
}

func (crawler *Crawler) Run(outsideContext context.Context, config *config.Config) error {
	if crawler.Workers == nil {
		return errors.New("no workers")
	}

	outsidectx, cancel := context.WithCancel(outsideContext)
	defer cancel()

	conn, err := storage.NewChConnection(config)
	if err != nil {
		return err
	}

	ctx := NewCrawlerContext(nil, conn, config)

	for _, worker := range crawler.Workers {
		go worker.Run(ctx, outsidectx)

		ctx.Clear()
	}

	select {
	case <-outsideContext.Done():
		return errors.ErrUnsupported

	}

	return nil
}
