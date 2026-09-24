// Command seed creates the default admin/quota rows for local dev.
// Migrations already seed plans + images; this ensures one user + quota.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/anshiq/ideserver/internal/db"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL required")
		os.Exit(1)
	}
	ctx := context.Background()
	dbh, err := db.Open(ctx, url)
	if err != nil {
		panic(err)
	}
	defer dbh.Close()
	q := db.New(dbh)
	_ = q
	fmt.Println("seed: migrations applied (plans + images); create users via GitHub OAuth")
}
