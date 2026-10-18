package storage

import (
	"database/sql/driver"

	"github.com/crispyLL/parltube-crawler/internal/config"
)

// crawl state storage DAO
type CrawlStateStorage struct {
	conn driver.Conn
}

func NewCrawlState(cfg *config.Config) *CrawlStateStorage {
	return &CrawlStateStorage{}
}

func (c *CrawlStateStorage) GetLastChannel() {

}
