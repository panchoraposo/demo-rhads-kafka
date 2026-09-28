package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Order struct {
	ID       int64   `json:"id"`
	Customer string  `json:"customer"`
	Amount   float64 `json:"amount"`
	Status   string  `json:"status"`
}

type DB struct {
	pool *pgxpool.Pool
}

// OpenFromEnv builds a pool from DATABASE_URL or DATABASE_HOST + user/password/name.
func OpenFromEnv() (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("DATABASE_HOST")
		if host == "" {
			return nil, fmt.Errorf("DATABASE_URL or DATABASE_HOST not set")
		}
		if !strings.Contains(host, ":") {
			host = host + ":" + env("DATABASE_PORT", "5432")
		}
		user := env("DATABASE_USER", "orders")
		pass := env("DATABASE_PASSWORD", "backstage")
		name := env("DATABASE_NAME", "orders")
		u := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(user, pass),
			Host:   host,
			Path:   "/" + name,
		}
		q := u.Query()
		q.Set("sslmode", env("DATABASE_SSLMODE", "disable"))
		u.RawQuery = q.Encode()
		dsn = u.String()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}

func (d *DB) Create(ctx context.Context, customer string, amount float64) (*Order, error) {
	var o Order
	err := d.pool.QueryRow(ctx,
		`INSERT INTO public.orders (customer, amount, status) VALUES ($1, $2, 'new')
		 RETURNING id, customer, amount::float8, status`,
		customer, amount,
	).Scan(&o.ID, &o.Customer, &o.Amount, &o.Status)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (d *DB) Get(ctx context.Context, id int64) (*Order, error) {
	var o Order
	err := d.pool.QueryRow(ctx,
		`SELECT id, customer, amount::float8, status FROM public.orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.Customer, &o.Amount, &o.Status)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (d *DB) UpdateStatus(ctx context.Context, id int64, status string) (before, after *Order, err error) {
	return d.Update(ctx, id, &status, nil)
}

// Update patches status and/or amount. At least one pointer must be non-nil.
func (d *DB) Update(ctx context.Context, id int64, status *string, amount *float64) (before, after *Order, err error) {
	if status == nil && amount == nil {
		return nil, nil, fmt.Errorf("nothing to update")
	}
	before, err = d.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	sets := make([]string, 0, 2)
	args := []any{id}
	n := 2
	if status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", n))
		args = append(args, *status)
		n++
	}
	if amount != nil {
		sets = append(sets, fmt.Sprintf("amount = $%d", n))
		args = append(args, *amount)
		n++
	}
	q := fmt.Sprintf(
		`UPDATE public.orders SET %s WHERE id = $1
		 RETURNING id, customer, amount::float8, status`,
		strings.Join(sets, ", "),
	)
	var o Order
	err = d.pool.QueryRow(ctx, q, args...).Scan(&o.ID, &o.Customer, &o.Amount, &o.Status)
	if err != nil {
		return before, nil, err
	}
	return before, &o, nil
}

func (d *DB) Delete(ctx context.Context, id int64) (*Order, error) {
	before, err := d.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	_, err = d.pool.Exec(ctx, `DELETE FROM public.orders WHERE id = $1`, id)
	if err != nil {
		return before, err
	}
	return before, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (o *Order) AsMap() map[string]any {
	if o == nil {
		return nil
	}
	return map[string]any{
		"id":       o.ID,
		"customer": o.Customer,
		"amount":   o.Amount,
		"status":   o.Status,
	}
}
