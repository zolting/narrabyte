package models

// DocChangedFile represents a documentation file modified during generation.
type DocChangedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// DocConversationMessage represents a single exchange between the user and assistant.
type DocConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DocGenerationResult captures the outcome of a documentation generation run.
type DocGenerationResult struct {
	Branch       string                   `json:"branch"`
	Files        []DocChangedFile         `json:"files"`
	Diff         string                   `json:"diff"`
	Summary      string                   `json:"summary"`
	Conversation []DocConversationMessage `json:"conversation"`
}
