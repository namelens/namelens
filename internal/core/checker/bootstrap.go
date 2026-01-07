package checker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultBootstrapURL = "https://data.iana.org/rdap/dns.json"

const (
	bootstrapMetaVersion     = "bootstrap_version"
	bootstrapMetaPublication = "bootstrap_publication"
	bootstrapMetaFetchedAt   = "bootstrap_fetched_at"
	bootstrapMetaSource      = "bootstrap_source"
)

// BootstrapStore provides persistence for bootstrap data.
type BootstrapStore interface {
	SetRDAPServers(ctx context.Context, tld string, servers []string, updatedAt time.Time) error
	GetRDAPServers(ctx context.Context, tld string) ([]string, error)
	SetBootstrapMeta(ctx context.Context, key, value string) error
	GetBootstrapMeta(ctx context.Context, key string) (string, error)
	CountBootstrapTLDs(ctx context.Context) (int, error)
}

// BootstrapService fetches and caches IANA RDAP bootstrap data.
type BootstrapService struct {
	Store      BootstrapStore
	HTTPClient *http.Client
	BaseURL    string
	Clock      func() time.Time
}

// BootstrapDocument represents the IANA RDAP DNS bootstrap response.
type BootstrapDocument struct {
	Version     string       `json:"version"`
	Publication string       `json:"publication"`
	Services    [][][]string `json:"services"`
}

// BootstrapSummary reports update results.
type BootstrapSummary struct {
	TLDCount    int
	Version     string
	Publication time.Time
	FetchedAt   time.Time
}

// BootstrapStatus reports cached bootstrap metadata.
type BootstrapStatus struct {
	TLDCount    int
	Version     string
	Publication time.Time
	FetchedAt   time.Time
	Source      string
}

// Update fetches bootstrap data from IANA and stores it.
func (b *BootstrapService) Update(ctx context.Context) (*BootstrapSummary, error) {
	if b == nil || b.Store == nil {
		return nil, errors.New("bootstrap store is not configured")
	}

	client := b.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	baseURL := strings.TrimSpace(b.BaseURL)
	if baseURL == "" {
		baseURL = defaultBootstrapURL
	}

	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build bootstrap request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch bootstrap data: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck // best-effort cleanup on HTTP response body

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bootstrap request failed: status %d", resp.StatusCode)
	}

	var doc BootstrapDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode bootstrap data: %w", err)
	}

	updatedAt := b.now()
	tldCount := 0

	for _, service := range doc.Services {
		if len(service) != 2 {
			continue
		}
		tlds := service[0]
		urls := service[1]
		if len(tlds) == 0 || len(urls) == 0 {
			continue
		}

		for _, tld := range tlds {
			if err := b.Store.SetRDAPServers(ctx, tld, urls, updatedAt); err != nil {
				return nil, err
			}
			tldCount++
		}
	}

	_ = b.Store.SetBootstrapMeta(ctx, bootstrapMetaVersion, doc.Version)
	_ = b.Store.SetBootstrapMeta(ctx, bootstrapMetaPublication, doc.Publication)
	_ = b.Store.SetBootstrapMeta(ctx, bootstrapMetaFetchedAt, updatedAt.Format(time.RFC3339))
	_ = b.Store.SetBootstrapMeta(ctx, bootstrapMetaSource, baseURL)

	publication := parseTime(doc.Publication)

	return &BootstrapSummary{
		TLDCount:    tldCount,
		Version:     doc.Version,
		Publication: publication,
		FetchedAt:   updatedAt,
	}, nil
}

// Status returns cached bootstrap metadata.
func (b *BootstrapService) Status(ctx context.Context) (*BootstrapStatus, error) {
	if b == nil || b.Store == nil {
		return nil, errors.New("bootstrap store is not configured")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	count, err := b.Store.CountBootstrapTLDs(ctx)
	if err != nil {
		return nil, err
	}

	version, err := b.Store.GetBootstrapMeta(ctx, bootstrapMetaVersion)
	if err != nil {
		return nil, err
	}

	publication, err := b.Store.GetBootstrapMeta(ctx, bootstrapMetaPublication)
	if err != nil {
		return nil, err
	}

	fetchedAt, err := b.Store.GetBootstrapMeta(ctx, bootstrapMetaFetchedAt)
	if err != nil {
		return nil, err
	}

	source, err := b.Store.GetBootstrapMeta(ctx, bootstrapMetaSource)
	if err != nil {
		return nil, err
	}

	return &BootstrapStatus{
		TLDCount:    count,
		Version:     version,
		Publication: parseTime(publication),
		FetchedAt:   parseTime(fetchedAt),
		Source:      source,
	}, nil
}

// LookupServers returns cached RDAP servers for the TLD.
func (b *BootstrapService) LookupServers(ctx context.Context, tld string) ([]string, error) {
	if b == nil || b.Store == nil {
		return nil, errors.New("bootstrap store is not configured")
	}
	return b.Store.GetRDAPServers(ctx, tld)
}

func (b *BootstrapService) now() time.Time {
	if b != nil && b.Clock != nil {
		return b.Clock()
	}
	return time.Now().UTC()
}

func parseTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}
