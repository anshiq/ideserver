package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type listenConn struct {
	conn *pgx.Conn
	ch   string
}

func newListenConn(ctx context.Context, url, channel string) (*listenConn, error) {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("db: listen connect: %w", err)
	}
	if _, err := conn.Exec(ctx, "LISTEN "+quoteIdent(channel)); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("db: listen: %w", err)
	}
	return &listenConn{conn: conn, ch: channel}, nil
}

func (l *listenConn) close() { _ = l.conn.Close(context.Background()) }

func (l *listenConn) loop(ctx context.Context, fn func(string)) error {
	for {
		n, err := l.conn.WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("db: notify wait: %w", err)
		}
		if n.Channel == l.ch {
			fn(n.Payload)
		}
	}
}

func quoteIdent(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for _, c := range []byte(s) {
		if c == '"' {
			out = append(out, '"')
		}
		out = append(out, c)
	}
	return string(append(out, '"'))
}
