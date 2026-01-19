package ailink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fulmenhq/gofulmen/schema"
	"github.com/namelens/namelens/internal/ailink/content"
	"github.com/namelens/namelens/internal/ailink/driver"
	"github.com/namelens/namelens/internal/ailink/prompt"
)

const (
	defaultPromptSlug = "name-availability"
	defaultTimeout    = 60 * time.Second
	maxTimeout        = 5 * time.Minute
)

// Service coordinates prompt loading, provider selection, and driver execution.
type Service struct {
	Providers *Registry
	Registry  prompt.Registry
	Catalog   *schema.Catalog
}

// Search runs an expert search using a role-selected provider.
func (s *Service) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if s == nil || s.Providers == nil {
		return nil, errors.New("ailink provider registry not configured")
	}
	if s.Registry == nil {
		return nil, errors.New("ailink prompt registry not configured")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	slug := strings.TrimSpace(req.PromptSlug)
	if slug == "" {
		slug = defaultPromptSlug
	}

	promptDef, err := s.Registry.Get(slug)
	if err != nil {
		return nil, err
	}

	systemPrompt, userPrompt, err := renderPrompt(promptDef, name, req.Depth)
	if err != nil {
		return nil, err
	}

	tools := promptTools(promptDef, req.UseTools)

	// search_parameters is an xAI-specific extension used to enable server-side web/X search.
	// Other providers (e.g. OpenAI) should run without search rather than failing.
	searchParams := buildSearchParams(promptDef.Config.Tools, req.UseTools)

	messages := []content.Message{
		{Role: "system", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: systemPrompt}}},
		{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: userPrompt}}},
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = slug
	}

	resolved, err := s.Providers.ResolveWithDepth(role, promptDef, req.Model, req.Depth)
	if err != nil {
		return nil, err
	}

	driverReq := &driver.Request{
		Model:            resolved.Model,
		Messages:         messages,
		Tools:            tools,
		SearchParameters: searchParams,
		ResponseFormat:   &driver.ResponseFormat{Type: "json_object"},
		PromptSlug:       promptDef.Config.Slug,
	}

	// search_parameters only works with the xAI driver. For other drivers, run “offline”.
	// Note: some prompts declare web_search/x_search tools; these are xAI-only.
	if resolved.Driver.Name() != "xai" {
		driverReq.SearchParameters = nil
		driverReq.Tools = nil
	}
	if driverReq.SearchParameters != nil {
		driverReq.Tools = nil // Prefer search_parameters for xAI; avoid conflicts
	}

	duration := s.Providers.cfg.DefaultTimeout
	if duration <= 0 {
		duration = defaultTimeout
	}
	if req.TimeoutSec > 0 {
		duration = time.Duration(req.TimeoutSec) * time.Second
	}
	if duration > maxTimeout {
		duration = maxTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	resp, err := resolved.Driver.Complete(ctx, driverReq)
	if err != nil {
		return nil, err
	}

	raw := extractContent(resp)
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("empty response content")
	}

	parsed, err := decodeSearchResponse([]byte(raw))
	if err != nil {
		return nil, &RawResponseError{Err: err, Raw: json.RawMessage(raw)}
	}

	if err := s.validateResponse(promptDef, []byte(raw)); err != nil {
		// Preserve parsed fields to keep CLI output useful, but still signal schema failure.
		parsed.Raw = append(parsed.Raw[:0], raw...)
		parsed.Raw = truncateJSONRaw(parsed.Raw, rawLimit(s.Providers.cfg))
		return parsed, &RawResponseError{Err: err, Raw: json.RawMessage(raw)}
	}

	if isRawCaptureEnabled(s.Providers.cfg, req.IncludeRaw) {
		parsed.Raw = append(parsed.Raw[:0], raw...)
		parsed.Raw = truncateJSONRaw(parsed.Raw, rawLimit(s.Providers.cfg))
	} else {
		parsed.Raw = nil
	}

	return parsed, nil
}

// Generate runs a generation prompt with arbitrary variables.
func (s *Service) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if s == nil || s.Providers == nil {
		return nil, errors.New("ailink provider registry not configured")
	}
	if s.Registry == nil {
		return nil, errors.New("ailink prompt registry not configured")
	}

	slug := strings.TrimSpace(req.PromptSlug)
	if slug == "" {
		return nil, errors.New("prompt slug is required")
	}

	promptDef, err := s.Registry.Get(slug)
	if err != nil {
		return nil, err
	}

	// Validate required variables
	for _, required := range promptDef.Config.Input.RequiredVariables {
		if val, ok := req.Variables[required]; !ok || strings.TrimSpace(val) == "" {
			return nil, fmt.Errorf("required variable %q not provided", required)
		}
	}

	depth := strings.TrimSpace(req.Depth)
	if depth == "" {
		depth = "quick"
	}

	systemPrompt, userPrompt, err := renderPromptWithVars(promptDef, req.Variables, depth)
	if err != nil {
		return nil, err
	}

	tools := promptTools(promptDef, req.UseTools)

	// search_parameters is an xAI-specific extension used to enable server-side web/X search.
	// Other providers (e.g. OpenAI) should run without search rather than failing.
	searchParams := buildSearchParams(promptDef.Config.Tools, req.UseTools)

	messages := []content.Message{
		{Role: "system", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: systemPrompt}}},
		{Role: "user", Content: []content.ContentBlock{{Type: content.ContentTypeText, Text: userPrompt}}},
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = slug
	}

	resolved, err := s.Providers.ResolveWithDepth(role, promptDef, req.Model, depth)
	if err != nil {
		return nil, err
	}

	driverReq := &driver.Request{
		Model:            resolved.Model,
		Messages:         messages,
		Tools:            tools,
		SearchParameters: searchParams,
		ResponseFormat:   &driver.ResponseFormat{Type: "json_object"},
		PromptSlug:       promptDef.Config.Slug,
	}

	// search_parameters only works with the xAI driver. For other drivers, run “offline”.
	// Note: some prompts declare web_search/x_search tools; these are xAI-only.
	if resolved.Driver.Name() != "xai" {
		driverReq.SearchParameters = nil
		driverReq.Tools = nil
	}
	if driverReq.SearchParameters != nil {
		driverReq.Tools = nil
	}

	duration := s.Providers.cfg.DefaultTimeout
	if duration <= 0 {
		duration = defaultTimeout
	}
	if req.TimeoutSec > 0 {
		duration = time.Duration(req.TimeoutSec) * time.Second
	}
	if duration > maxTimeout {
		duration = maxTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	resp, err := resolved.Driver.Complete(ctx, driverReq)
	if err != nil {
		return nil, err
	}

	raw := extractContent(resp)
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("empty response content")
	}

	if err := s.validateResponse(promptDef, []byte(raw)); err != nil {
		return nil, &RawResponseError{Err: err, Raw: json.RawMessage(raw)}
	}

	response := &GenerateResponse{Raw: json.RawMessage(raw)}
	if isRawCaptureEnabled(s.Providers.cfg, req.IncludeRaw) {
		response.Raw = truncateJSONRaw(response.Raw, rawLimit(s.Providers.cfg))
	}

	return response, nil
}

func promptTools(def *prompt.Prompt, enabled bool) []driver.Tool {
	if def == nil || !enabled {
		return nil
	}
	if len(def.Config.Tools) == 0 {
		return nil
	}

	tools := make([]driver.Tool, 0, len(def.Config.Tools))
	for _, tool := range def.Config.Tools {
		tools = append(tools, driver.Tool{Type: tool.Type, Config: tool.Config})
	}
	return tools
}

// buildSearchParams maps search-specific tools to xAI search_parameters.
func buildSearchParams(tools []prompt.ToolConfig, enabled bool) *driver.SearchParameters {
	if !enabled || len(tools) == 0 {
		return nil
	}
	params := &driver.SearchParameters{
		Mode:            "auto",
		ReturnCitations: true,
	}
	sourceMap := map[string]string{
		"web_search":  "web",
		"x_search":    "x",
		"live_search": "web", // fallback
	}
	for _, tool := range tools {
		if srcType, ok := sourceMap[tool.Type]; ok {
			params.Sources = append(params.Sources, driver.Source{Type: srcType})
		}
	}
	if len(params.Sources) == 0 {
		return nil
	}
	return params
}

func renderPrompt(def *prompt.Prompt, name, depth string) (string, string, error) {
	if def == nil {
		return "", "", errors.New("prompt is required")
	}
	vars := map[string]string{
		"name":  name,
		"input": name,
	}
	if depth != "" {
		vars["depth"] = depth
	}

	system := applyVars(def.Config.SystemTemplate, vars)
	user := ""
	if depth != "" {
		if variant, ok := def.Config.DepthVariants[depth]; ok {
			user = variant
		}
	}
	if user == "" {
		user = def.Config.UserTemplate
	}
	if user == "" {
		user = "{{input}}"
	}
	user = applyVars(user, vars)

	if strings.TrimSpace(system) == "" {
		return "", "", errors.New("system prompt is required")
	}
	return system, user, nil
}

func applyVars(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

// applyConditionals handles {{#if var}}content{{else}}fallback{{/if}} blocks.
// If the variable exists and is non-empty, the content is included; otherwise the fallback is used.
func applyConditionals(template string, vars map[string]string) string {
	result := template
	for {
		start := strings.Index(result, "{{#if")
		if start == -1 {
			break
		}
		tagEnd := strings.Index(result[start:], "}}")
		if tagEnd == -1 {
			break
		}
		tagEnd += start

		varName := strings.TrimSpace(result[start+len("{{#if") : tagEnd])
		blockStart := tagEnd + 2

		elseStart, elseEnd, endStart, endEnd := findConditionalBlock(result, blockStart)
		if endStart == -1 {
			break
		}

		ifContent := result[blockStart:endStart]
		elseContent := ""
		if elseStart != -1 {
			ifContent = result[blockStart:elseStart]
			elseContent = result[elseEnd:endStart]
		}

		value, exists := vars[varName]
		replacement := elseContent
		if exists && strings.TrimSpace(value) != "" {
			replacement = ifContent
		}

		result = result[:start] + replacement + result[endEnd:]
	}
	return result
}

func findConditionalBlock(input string, start int) (int, int, int, int) {
	depth := 0
	elseStart := -1
	elseEnd := -1

	pos := start
	for {
		openIdx := strings.Index(input[pos:], "{{")
		if openIdx == -1 {
			return -1, -1, -1, -1
		}
		openIdx += pos

		closeIdx := strings.Index(input[openIdx:], "}}")
		if closeIdx == -1 {
			return -1, -1, -1, -1
		}
		closeIdx += openIdx

		tag := strings.TrimSpace(input[openIdx+2 : closeIdx])
		switch {
		case tag == "#if" || strings.HasPrefix(tag, "#if "):
			depth++
		case tag == "/if":
			if depth == 0 {
				return elseStart, elseEnd, openIdx, closeIdx + 2
			}
			depth--
		case tag == "else" && depth == 0 && elseStart == -1:
			elseStart = openIdx
			elseEnd = closeIdx + 2
		}

		pos = closeIdx + 2
	}
}

// renderPromptWithVars renders a prompt template with arbitrary variables and conditionals.
func renderPromptWithVars(def *prompt.Prompt, vars map[string]string, depth string) (string, string, error) {
	if def == nil {
		return "", "", errors.New("prompt is required")
	}

	// Apply conditionals first, then variable substitution
	system := applyConditionals(def.Config.SystemTemplate, vars)
	system = applyVars(system, vars)

	user := ""
	if depth != "" {
		if variant, ok := def.Config.DepthVariants[depth]; ok {
			user = variant
		}
	}
	if user == "" {
		user = def.Config.UserTemplate
	}
	if user == "" {
		user = "{{concept}}" // Default for generate prompts
	}
	user = applyConditionals(user, vars)
	user = applyVars(user, vars)

	if strings.TrimSpace(system) == "" {
		return "", "", errors.New("system prompt is required")
	}
	return system, user, nil
}

func extractContent(resp *driver.Response) string {
	if resp == nil {
		return ""
	}
	if len(resp.Content) == 0 {
		return ""
	}
	parts := make([]string, 0, len(resp.Content))
	for _, block := range resp.Content {
		parts = append(parts, block.Text)
	}
	return strings.Join(parts, "\n")
}

func decodeSearchResponse(raw []byte) (*SearchResponse, error) {
	var parsed SearchResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if isEmptySearchResponse(parsed) {
		parsed.Raw = append(parsed.Raw[:0], raw...)
	}
	return &parsed, nil
}

func isEmptySearchResponse(resp SearchResponse) bool {
	if strings.TrimSpace(resp.Summary) != "" {
		return false
	}
	if strings.TrimSpace(resp.RiskLevel) != "" {
		return false
	}
	if resp.LikelyAvailable != nil {
		return false
	}
	if resp.Confidence != nil {
		return false
	}
	if len(resp.Insights) > 0 {
		return false
	}
	if len(resp.Mentions) > 0 {
		return false
	}
	if len(resp.Recommendations) > 0 {
		return false
	}
	return true
}

func (s *Service) validateResponse(def *prompt.Prompt, payload []byte) error {
	if def == nil {
		return nil
	}
	if len(def.Config.ResponseSchema) == 0 {
		return nil
	}
	if ref, ok := def.Config.ResponseSchema["$ref"].(string); ok && ref != "" {
		catalog := s.Catalog
		if catalog == nil {
			return errors.New("schema catalog not configured")
		}
		diagnostics, err := catalog.ValidateDataByID(ref, payload)
		if err != nil {
			return err
		}
		if len(diagnostics) > 0 {
			return fmt.Errorf("response schema validation failed: %s", diagnostics[0].Message)
		}
		return nil
	}

	schemaBytes, err := json.Marshal(def.Config.ResponseSchema)
	if err != nil {
		return fmt.Errorf("encode response schema: %w", err)
	}
	validator, err := schema.NewValidator(schemaBytes)
	if err != nil {
		return fmt.Errorf("compile response schema: %w", err)
	}
	diagnostics, err := validator.ValidateJSON(payload)
	if err != nil {
		return err
	}
	if len(diagnostics) > 0 {
		return fmt.Errorf("response schema validation failed: %s", diagnostics[0].Message)
	}
	return nil
}
