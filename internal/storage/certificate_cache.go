// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/acme/autocert"
)

// NewCertificateCache creates an cache instance that can be used with autocert.Cache.
// It returns any errors that could happen while connecting to SQL.
func (s *Storage) NewCertificateCache() *CertificateCache {
	return &CertificateCache{db: s.db}
}

// Making sure that we're adhering to the autocert.Cache interface.
var _ autocert.Cache = (*CertificateCache)(nil)

// CertificateCache provides a SQL backend to the autocert cache.
type CertificateCache struct {
	db *pgxpool.Pool
}

// Get returns a certificate data for the specified key.
// If there's no such key, Get returns ErrCacheMiss.
func (c *CertificateCache) Get(ctx context.Context, key string) ([]byte, error) {
	rows, _ := c.db.Query(ctx,
		`SELECT data::bytea FROM acme_cache WHERE key = $1`, key)
	data, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[[]byte])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, autocert.ErrCacheMiss
	} else if err != nil {
		return nil, fmt.Errorf("storage: select from acme_cache: %w", err)
	}
	return data, nil
}

// Put stores the data in the cache under the specified key.
func (c *CertificateCache) Put(ctx context.Context, key string, data []byte,
) error {
	_, err := c.db.Exec(ctx, `
INSERT INTO acme_cache (key, data,      updated_at)
                VALUES ($1,  $2::bytea, now())
ON CONFLICT (key) DO
  UPDATE SET data = $2::bytea, updated_at = now()`,
		key, data)
	if err != nil {
		return fmt.Errorf("storage: update acme_cache: %w", err)
	}
	return nil
}

// Delete removes a certificate data from the cache under the specified key.
// If there's no such key in the cache, Delete returns nil.
func (c *CertificateCache) Delete(ctx context.Context, key string) error {
	_, err := c.db.Exec(ctx, `DELETE FROM acme_cache WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("storage: delete from acme_cache: %w", err)
	}
	return nil
}
