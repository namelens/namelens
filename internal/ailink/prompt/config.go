package prompt

// Config describes a prompt definition loaded from YAML.
type Config struct {
	Slug           string            `yaml:"slug" json:"slug"`
	Name           string            `yaml:"name,omitempty" json:"name,omitempty"`
	Description    string            `yaml:"description,omitempty" json:"description,omitempty"`
	Version        string            `yaml:"version,omitempty" json:"version,omitempty"`
	Author         string            `yaml:"author,omitempty" json:"author,omitempty"`
	Updated        string            `yaml:"updated,omitempty" json:"updated,omitempty"`
	Input          InputSpec         `yaml:"input,omitempty" json:"input,omitempty"`
	SystemTemplate string            `yaml:"system_template,omitempty" json:"system_template,omitempty"`
	UserTemplate   string            `yaml:"user_template,omitempty" json:"user_template,omitempty"`
	DepthVariants  map[string]string `yaml:"depth_variants,omitempty" json:"depth_variants,omitempty"`
	Tools          []ToolConfig      `yaml:"tools,omitempty" json:"tools,omitempty"`
	ResponseSchema map[string]any    `yaml:"response_schema,omitempty" json:"response_schema,omitempty"`
	ResponseOpts   map[string]any    `yaml:"response_options,omitempty" json:"response_options,omitempty"`
	ProviderHints  map[string]any    `yaml:"provider_hints,omitempty" json:"provider_hints,omitempty"`
}

// InputSpec defines prompt input requirements.
type InputSpec struct {
	RequiredVariables []string `yaml:"required_variables,omitempty" json:"required_variables,omitempty"`
	OptionalVariables []string `yaml:"optional_variables,omitempty" json:"optional_variables,omitempty"`
	AcceptsImages     bool     `yaml:"accepts_images,omitempty" json:"accepts_images,omitempty"`
	ImageTypes        []string `yaml:"image_types,omitempty" json:"image_types,omitempty"`
	MaxImages         int      `yaml:"max_images,omitempty" json:"max_images,omitempty"`
}

// ToolConfig represents a server-side tool configuration in the prompt.
type ToolConfig struct {
	Type   string         `yaml:"type" json:"type"`
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// Prompt wraps a validated prompt configuration with its source.
type Prompt struct {
	Config Config
	Source string
}
