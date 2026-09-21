// Package app wires the crawler components together and runs them.
package crawler

import (
	"sync"

	"github.com/crispyLL/parltube-crawler/internal/storage"
)

type IRevolvingProxy interface {
	Next() int
}

// what we give to each worker
type CrawlerContext struct {
	GlobalStopChannel <-chan struct{}
	Proxies           IRevolvingProxy // struct that revolves Mullvad proxies, abstraction
}

// Run starts the crawler and blocks until ctx is cancelled.
type Crawler struct {
	MaxWorkerCount     uint
	CurrentWorkerCount uint
	once               sync.Once
	lock               sync.Mutex
	Context            CrawlerContext
	Workers            []*Worker
	Repo               *storage.ExtDataStorage
}

func NewCrawler(MaxWorkerCount uint, Repo *storage.ExtDataStorage) *Crawler {
	return &Crawler{
		MaxWorkerCount: MaxWorkerCount,
		Workers:        make([]*Worker, MaxWorkerCount),
	}
}

func (crawler *Crawler) Run() {
	if crawler.Workers == nil {
		return
	}

	go func() {

	}()
}
