package supabase

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps a pgxpool.Pool. It exists so callers (like internal/adapters/httpapi)
// can depend on a small interface (Ping) instead of importing pgx directly.
//
// Connect against the direct connection string or the session-mode pooler,
// not the transaction-mode pooler — transaction mode doesn't support
// LISTEN/NOTIFY, which ADR-0004's notification-recovery item needs. Direct
// connections are IPv6-only without Supabase's paid IPv4 add-on; use the
// session pooler (IPv4) when the deploy environment lacks outbound IPv6.
type Pool struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, connString string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &Pool{pool: pool}, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Pool) Close() {
	p.pool.Close()
}
