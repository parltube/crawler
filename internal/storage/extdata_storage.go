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
)

type BatchedStorage[T any] struct {
	connection   driver.Conn
	interval     time.Duration
	adder        chan T
	currentSql   string
	MaxBatchSize uint
}

func NewBatchedStorage[T any](maxBatchSize uint) (*BatchedStorage[T], error) {
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
	return &BatchedStorage[T]{
		connection: conn,
		adder:      make(chan T, maxBatchSize),
	}, nil
}

// collects data into the bulk asynchronusly inserts to keep clootting the disk with small inserts clickhouse, every 20-30 seconds.
// i think i will do it thorugh channels
func (stor *BatchedStorage[T]) Insert(data T, query string) {
	stor.adder <- data
}

func (stor *BatchedStorage[T]) Run(ctx context.Context) {

}

func (b *Batcher[T]) Add(v T) { b.in <- v }

func (b *Batcher[T]) Run(ctx context.Context) {
	buf := make([]T, 0, b.maxRows)
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
			log.Printf("flush %s: %v (dropped %d rows)", b.query, err, len(buf))
		}
		buf = buf[:0]
	}

	for {
		select {
		case v := <-b.in:
			buf = append(buf, v)
			if len(buf) >= b.maxRows {
				flush()
			}
		case <-t.C:
			flush()
		case <-ctx.Done():
			// дочитать то, что уже в канале, и сбросить
			for {
				select {
				case v := <-b.in:
					buf = append(buf, v)
				default:
					flush()
					return
				}
			}
		}
	}
}

func (b *Batcher[T]) send(ctx context.Context, rows []T) error {
	batch, err := b.conn.PrepareBatch(ctx, b.query)
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
