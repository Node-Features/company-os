// Command migrate applies supabase/migrations/*.sql directly against
// DATABASE_URL, in identity order. Used because the Supabase CLI project
// is not linked to a hosted ref yet (supabase/README.md); once it is,
// `supabase db push` can replace this. Not idempotent — safe to re-run
// only against a fresh database, since it does not track which migrations
// already applied.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Println("DATABASE_URL not set")
		os.Exit(1)
	}

	dir := "../../supabase/migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("read migrations dir:", err)
		os.Exit(1)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}
	defer pool.Close()

	for _, name := range files {
		path := filepath.Join(dir, name)
		sql, err := os.ReadFile(path)
		if err != nil {
			fmt.Println("read file:", err)
			os.Exit(1)
		}
		fmt.Println("applying", name)
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			fmt.Println("FAILED applying", name, ":", err)
			os.Exit(1)
		}
		fmt.Println("OK", name)
	}
	fmt.Println("ALL MIGRATIONS APPLIED")
}
