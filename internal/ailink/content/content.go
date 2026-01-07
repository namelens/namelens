package content

// ContentType represents supported content types using IANA media types.
type ContentType string

const (
	ContentTypeText ContentType = "text/plain"
	ContentTypeJSON ContentType = "application/json"
)

// ContentBlock represents a single piece of content.
type ContentBlock struct {
	Type    ContentType `json:"type"`
	Text    string      `json:"text,omitempty"`
	Data    []byte      `json:"data,omitempty"`
	DataURL string      `json:"data_url,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}
