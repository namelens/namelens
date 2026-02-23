package ailink

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fulmenhq/gofulmen/schema"
	"github.com/stretchr/testify/require"

	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

type recordingDriver struct {
	name string
	req  *driver.Request
}

func (d *recordingDriver) Complete(ctx context.Context, req *driver.Request) (*driver.Response, error) {
	d.req = req
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
