package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SetRDAPServers stores RDAP server URLs for a TLD.
func (s *Store) SetRDAPServers(ctx context.Context, tld string, servers []string, updatedAt time.Time) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	normalized := normalizeTLD(tld)
	if normalized == "" {
		return errors.New("tld is required")
	}

	payload, err := json.Marshal(servers)
	if err != nil {
		return fmt.Errorf("marshal rdap servers: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO bootstrap_tlds (tld, rdap_urls, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(tld) DO UPDATE SET
			rdap_urls = excluded.rdap_urls,
			updated_at = excluded.updated_at
	`, normalized, string(payload), updatedAt.Unix())
	if err != nil {
		return fmt.Errorf("store rdap servers: %w", err)
	}

	return nil
}

// GetRDAPServers returns RDAP server URLs for a TLD.
func (s *Store) GetRDAPServers(ctx context.Context, tld string) ([]string, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	normalized := normalizeTLD(tld)
	if normalized == "" {
		return nil, errors.New("tld is required")
	}

	var payload string
	if err := s.DB.QueryRowContext(ctx, `SELECT rdap_urls FROM bootstrap_tlds WHERE tld = ?`, normalized).Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch rdap servers: %w", err)
	}

	var servers []string
	if err := json.Unmarshal([]byte(payload), &servers); err != nil {
		return nil, fmt.Errorf("decode rdap servers: %w", err)
	}

	return servers, nil
}

// SetBootstrapMeta stores a bootstrap metadata key/value.
func (s *Store) SetBootstrapMeta(ctx context.Context, key, value string) error {
	if s == nil || s.DB == nil {
		return errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if strings.TrimSpace(key) == "" {
		return errors.New("bootstrap meta key is required")
	}

	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO bootstrap_meta (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("store bootstrap meta: %w", err)
	}

	return nil
}

// GetBootstrapMeta returns a bootstrap metadata value.
func (s *Store) GetBootstrapMeta(ctx context.Context, key string) (string, error) {
	if s == nil || s.DB == nil {
		return "", errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if strings.TrimSpace(key) == "" {
		return "", errors.New("bootstrap meta key is required")
	}

	var value string
	if err := s.DB.QueryRowContext(ctx, `SELECT value FROM bootstrap_meta WHERE key = ?`, key).Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("fetch bootstrap meta: %w", err)
	}

	return value, nil
}

// CountBootstrapTLDs returns the number of cached TLD mappings.
func (s *Store) CountBootstrapTLDs(ctx context.Context) (int, error) {
	if s == nil || s.DB == nil {
		return 0, errors.New("store is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	var count int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM bootstrap_tlds`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count bootstrap tlds: %w", err)
	}

	return count, nil
}

func normalizeTLD(tld string) string {
	value := strings.ToLower(strings.TrimSpace(tld))
	value = strings.TrimPrefix(value, ".")
	return value
}
