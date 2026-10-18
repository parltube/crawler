package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/crispyLL/parltube-crawler/internal/config"
)

type BatchedStorage[T any] struct {
	connection   driver.Conn
	interval     time.Duration
	adder        chan T
	currentSql   string
	MaxBatchSize int
}

func NewChConnection(config *config.Config) (driver.Conn, error) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"clickhouse:9440"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "default",
			},
			ClientInfo: clickhouse.ClientInfo{
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: "an-example-go-client", Version: "0.1"},
				},
			},
			Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})),
			TLS: &tls.Config{
				InsecureSkipVerify: true,
			},
		})
	)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}

	return conn, nil
}

func NewBatchedStorage[T any](connection driver.Conn, config *config.Config) *BatchedStorage[T] {
	return &BatchedStorage[T]{
		connection: connection,
		adder:      make(chan T, config.BatchSize),
	}
}

// collects data into the bulk asynchronusly inserts to keep clootting the disk with small inserts clickhouse, every 20-30 seconds.
// i think i will do it thorugh channels
func (stor *BatchedStorage[T]) Insert(data T, query string) {
	stor.adder <- data
}

func (b *BatchedStorage[T]) Run(ctx context.Context) {
	buf := make([]T, 0, b.MaxBatchSize)
	t := time.NewTicker(b.interval)
	defer t.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		// отдельный контекст, чтобы финальный flush прошёл и при отмене ctx
		fctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := b.send(fctx, buf); err != nil {
			log.Printf("flush %s: %v (dropped %d rows)", b.currentSql, err, len(buf))
		}
		buf = buf[:0]
	}

	for {
		select {
		case v := <-b.adder:
			buf = append(buf, v)
			if len(buf) >= b.MaxBatchSize {
				flush()
			}
		case <-t.C:
			flush()
		case <-ctx.Done():
			// дочитать то, что уже в канале, и сбросить
			for {
				select {
				case v := <-b.adder:
					buf = append(buf, v)
				default:
					flush()
					return
				}
			}
		}
	}
}

func (b *BatchedStorage[T]) send(ctx context.Context, rows []T) error {
	batch, err := b.connection.PrepareBatch(ctx, b.currentSql)
	if err != nil {
		return err
	}
	for i := range rows {
		if err := batch.AppendStruct(&rows[i]); err != nil {
			batch.Abort()
			return err
		}
	}
	return batch.Send()
}
