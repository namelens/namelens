package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memoryBootstrapStore struct {
	servers map[string][]string
	meta    map[string]string
}

func (m *memoryBootstrapStore) SetRDAPServers(ctx context.Context, tld string, servers []string, updatedAt time.Time) error {
	if m.servers == nil {
		m.servers = make(map[string][]string)
	}
	m.servers[tld] = servers
	return nil
}

func (m *memoryBootstrapStore) GetRDAPServers(ctx context.Context, tld string) ([]string, error) {
	if m.servers == nil {
		return nil, nil
	}
	return m.servers[tld], nil
}

func (m *memoryBootstrapStore) SetBootstrapMeta(ctx context.Context, key, value string) error {
	if m.meta == nil {
		m.meta = make(map[string]string)
	}
	m.meta[key] = value
	return nil
}

func (m *memoryBootstrapStore) GetBootstrapMeta(ctx context.Context, key string) (string, error) {
	if m.meta == nil {
		return "", nil
	}
	return m.meta[key], nil
}

func (m *memoryBootstrapStore) CountBootstrapTLDs(ctx context.Context) (int, error) {
	return len(m.servers), nil
}

func TestBootstrapUpdate(t *testing.T) {
	payload := `{
  "version": "1.0",
  "publication": "2024-12-01T00:00:00Z",
  "services": [
    [["com", "net"], ["https://rdap.example.com/"]]
  ]
}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	store := &memoryBootstrapStore{}
	service := &BootstrapService{
		Store:      store,
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Clock: func() time.Time {
			return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	}

	summary, err := service.Update(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, summary.TLDCount)
	require.Equal(t, "1.0", summary.Version)
	require.Equal(t, time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), summary.Publication)
}

func TestBootstrapStatus(t *testing.T) {
	store := &memoryBootstrapStore{
		servers: map[string][]string{"com": {"https://rdap.example.com/"}},
		meta: map[string]string{
			bootstrapMetaVersion:     "1.0",
			bootstrapMetaPublication: "2024-12-01T00:00:00Z",
			bootstrapMetaFetchedAt:   "2025-01-01T00:00:00Z",
			bootstrapMetaSource:      "https://data.iana.org/rdap/dns.json",
		},
	}

	service := &BootstrapService{Store: store}
	status, err := service.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, status.TLDCount)
	require.Equal(t, "1.0", status.Version)
	require.Equal(t, time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), status.Publication)
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), status.FetchedAt)
}
