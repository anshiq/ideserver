package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartNotifyBridge LISTENs on instance_events and publishes payloads
// ("{id}:{state}") to the SSE hub. It reconnects with backoff; a missed
// NOTIFY is covered by the dashboard's refetch on reconnect (plan §1.5).
func StartNotifyBridge(ctx context.Context, pool *pgxpool.Pool, hub *Hub, log *slog.Logger) {
	backoff := time.Second
	for {
		if err := bridgeOnce(ctx, pool, hub); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn("notify bridge dropped, reconnecting", "err", err, "in", backoff.String())
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}
		backoff = time.Second
	}
}

func bridgeOnce(ctx context.Context, pool *pgxpool.Pool, hub *Hub) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `LISTEN "instance_events"`); err != nil {
		return err
	}
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if n.Channel == "instance_events" {
			hub.Publish(n.Payload)
		}
	}
}
