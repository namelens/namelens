package ailink

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/fulmenhq/gofulmen/schema"
	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

type recordingDriver struct {
	name        string
	req         *driver.Request
	hadDeadline bool
	deadline    time.Time
}

func (d *recordingDriver) Complete(ctx context.Context, req *driver.Request) (*driver.Response, error) {
	d.req = req
	d.deadline, d.hadDeadline = ctx.Deadline()
	return &driver.Response{Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: `{"summary":"ok"}`}}}, nil
}

func (d *recordingDriver) Name() string { return d.name }

func (d *recordingDriver) Capabilities() driver.Capabilities { return driver.Capabilities{} }

func TestServiceSearchDropsSearchParametersForNonXAI(t *testing.T) {
	drv := &recordingDriver{name: "openai"}

	providers := &Registry{cfg: Config{}}
	providers.cfg.DefaultProvider = "p"
	providers.cfg.Providers = map[string]ProviderInstanceConfig{
		"p": {
			Enabled:     true,
			AIProvider:  "openai",
			Models:      map[string]string{"default": "m"},
			Credentials: []CredentialConfig{{APIKey: "k"}},
		},
	}
	// Registry caches drivers by providerID:credKey. With no credential label and default priority,
	// selectCredential() uses "p0".
	providers.drivers = map[string]driver.Driver{"p:p0": drv}

	promptDef := &prompt.Prompt{Config: prompt.Config{Slug: "name-availability", SystemTemplate: "sys", UserTemplate: "usr", Tools: []prompt.ToolConfig{{Type: "web_search"}}}}
	svc := &Service{Providers: providers, Registry: stubPromptRegistry{prompt: promptDef}}

	resp, err := svc.Search(context.Background(), SearchRequest{Name: "test", PromptSlug: "name-availability", UseTools: true})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, drv.req)
	require.Nil(t, drv.req.SearchParameters)
	require.Nil(t, drv.req.Tools)
}

func TestServiceGenerateRejectsShortDomainFinderWithoutXAI(t *testing.T) {
	drv := &recordingDriver{name: "openai"}

	providers := &Registry{cfg: Config{}}
	providers.cfg.Providers = map[string]ProviderInstanceConfig{
		"namelens-openai": {
			Enabled:     true,
			AIProvider:  "openai",
			Models:      map[string]string{"default": "gpt-4o"},
			Credentials: []CredentialConfig{{APIKey: "k"}},
		},
		"namelens-xai": {
			Enabled:     true,
			AIProvider:  "xai",
			Models:      map[string]string{"default": "grok-4-1-fast-reasoning"},
			Credentials: []CredentialConfig{{APIKey: "k"}},
		},
	}
	providers.cfg.DefaultProvider = "namelens-openai"
	providers.drivers = map[string]driver.Driver{"namelens-openai:p0": drv}

	promptDef := &prompt.Prompt{Config: prompt.Config{
		Slug:           "short-domain-finder",
		SystemTemplate: "sys",
		UserTemplate:   "{{name}}",
		Input: prompt.InputSpec{
			RequiredVariables: []string{"name"},
		},
		Tools:          []prompt.ToolConfig{{Type: "web_search"}},
		ResponseSchema: map[string]any{"$ref": "ailink/v0/short-domain-finder-response"},
	}}
	svc := &Service{Providers: providers, Registry: stubPromptRegistry{prompt: promptDef}}

	resp, err := svc.Generate(context.Background(), GenerateRequest{
		PromptSlug: "short-domain-finder",
		Variables:  map[string]string{"name": "qswrangler"},
		UseTools:   true,
	})
	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), `prompt "short-domain-finder" requires web_search tool support`)
	require.Contains(t, err.Error(), `selected provider "namelens-openai"`)
	require.Contains(t, err.Error(), `use --provider namelens-xai`)
	require.Nil(t, drv.req, "request should fail before provider call")
}

type stubPromptRegistry struct {
	prompt *prompt.Prompt
}

func (s stubPromptRegistry) Get(slug string) (*prompt.Prompt, error) { return s.prompt, nil }
func (s stubPromptRegistry) List() []*prompt.Prompt                  { return []*prompt.Prompt{s.prompt} }

func TestServiceValidateResponseBrandPlanSchemaRef(t *testing.T) {
	svc := &Service{Catalog: schema.NewCatalog(filepath.Join("..", "..", "schemas"))}
	def := &prompt.Prompt{
		Config: prompt.Config{
			Slug:           "brand-plan",
			ResponseSchema: map[string]any{"$ref": "ailink/v0/brand-plan-response"},
		},
	}

	err := svc.validateResponse(def, []byte(`{"summary":"launch plan ready"}`))
	require.NoError(t, err)
}

func TestServiceValidateResponseBrandPlanSchemaRefRejectsMissingNestedFields(t *testing.T) {
	svc := &Service{Catalog: schema.NewCatalog(filepath.Join("..", "..", "schemas"))}
	def := &prompt.Prompt{
		Config: prompt.Config{
			Slug:           "brand-plan",
			ResponseSchema: map[string]any{"$ref": "ailink/v0/brand-plan-response"},
		},
	}

	invalid := `{
		"summary":"launch plan ready",
		"mentions":[
			{"source":"web"}
		]
	}`

	err := svc.validateResponse(def, []byte(invalid))
	require.Error(t, err)
	require.Contains(t, err.Error(), "response schema validation failed")
}

func TestServiceValidateResponseSearchBulkSchemaRef(t *testing.T) {
	svc := &Service{Catalog: schema.NewCatalog(filepath.Join("..", "..", "schemas"))}
	def := &prompt.Prompt{
		Config: prompt.Config{
			Slug:           "name-availability-bulk",
			ResponseSchema: map[string]any{"$ref": "ailink/v0/search-bulk-response"},
		},
	}

	valid := `{
		"items":[
			{"name":"alpha","summary":"clear","risk_level":"low"}
		]
	}`
	err := svc.validateResponse(def, []byte(valid))
	require.NoError(t, err)
}

// timeoutSearchHarness assembles a Service backed by recordingDriver so the
// timeout-propagation tests can inspect the context.Deadline the driver
// observes. The fixture mirrors TestServiceSearchDropsSearchParametersForNonXAI
// shape; only the driver's recording fields are interrogated.
func timeoutSearchHarness(t *testing.T, configDefault time.Duration) (*Service, *recordingDriver) {
	t.Helper()
	drv := &recordingDriver{name: "openai"}
	providers := &Registry{cfg: Config{DefaultProvider: "p", DefaultTimeout: configDefault}}
	providers.cfg.Providers = map[string]ProviderInstanceConfig{
		"p": {
			Enabled:     true,
			AIProvider:  "openai",
			Models:      map[string]string{"default": "m"},
			Credentials: []CredentialConfig{{APIKey: "k"}},
		},
	}
	providers.drivers = map[string]driver.Driver{"p:p0": drv}
	promptDef := &prompt.Prompt{Config: prompt.Config{
		Slug:           "name-availability",
		SystemTemplate: "sys",
		UserTemplate:   "usr",
	}}
	return &Service{Providers: providers, Registry: stubPromptRegistry{prompt: promptDef}}, drv
}

func TestServiceSearchHonorsRequestTimeoutSec(t *testing.T) {
	svc, drv := timeoutSearchHarness(t, 60*time.Second)

	before := time.Now()
	_, err := svc.Search(context.Background(), SearchRequest{
		Name:       "test",
		PromptSlug: "name-availability",
		TimeoutSec: 5,
	})
	require.NoError(t, err)

	require.True(t, drv.hadDeadline, "driver context should have a deadline")
	remaining := time.Until(drv.deadline)
	// Deadline should be ~5s out (the request's TimeoutSec), not ~60s (the
	// config default). Allow a 1s slack for test execution overhead in
	// either direction.
	require.LessOrEqual(t, remaining, 6*time.Second, "deadline farther than request timeout suggests override didn't apply")
	require.GreaterOrEqual(t, remaining, 4*time.Second, "deadline closer than request timeout suggests aggressive clamp")
	// Sanity-check that the deadline was set after our before-call timestamp.
	require.True(t, drv.deadline.After(before), "deadline must be in the future relative to test start")
}

func TestServiceSearchFallsBackToConfigDefaultWhenTimeoutSecZero(t *testing.T) {
	// Pick a config default that's distinguishable from the package
	// defaultTimeout constant (60s) so a regression that loses the config
	// path is caught.
	svc, drv := timeoutSearchHarness(t, 30*time.Second)

	_, err := svc.Search(context.Background(), SearchRequest{
		Name:       "test",
		PromptSlug: "name-availability",
		TimeoutSec: 0,
	})
	require.NoError(t, err)

	require.True(t, drv.hadDeadline)
	remaining := time.Until(drv.deadline)
	require.LessOrEqual(t, remaining, 31*time.Second, "deadline should track config default, not package fallback")
	require.GreaterOrEqual(t, remaining, 29*time.Second, "deadline should be ~config default")
}

func TestServiceGenerateHonorsRequestTimeoutSec(t *testing.T) {
	svc, drv := timeoutSearchHarness(t, 60*time.Second)

	_, err := svc.Generate(context.Background(), GenerateRequest{
		PromptSlug: "name-availability",
		Variables:  map[string]string{"name": "alpha"},
		TimeoutSec: 7,
	})
	require.NoError(t, err)

	require.True(t, drv.hadDeadline)
	remaining := time.Until(drv.deadline)
	require.LessOrEqual(t, remaining, 8*time.Second)
	require.GreaterOrEqual(t, remaining, 6*time.Second)
}
