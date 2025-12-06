package models

// DocChangedFile represents a documentation file modified during generation.
type DocChangedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// DocGenerationResult captures the outcome of a documentation generation run.
type DocGenerationResult struct {
	SessionID      uint             `json:"sessionId"`
	SessionKey     string           `json:"sessionKey"`
	Branch         string           `json:"branch"`
	TargetBranch   string           `json:"targetBranch"`
	DocsBranch     string           `json:"docsBranch"`
	DocsInCodeRepo bool             `json:"docsInCodeRepo"`
	Files          []DocChangedFile `json:"files"`
	Diff           string           `json:"diff"`
	Summary        string           `json:"summary"`
	ChatMessages   []ChatMessage    `json:"chatMessages,omitempty"`
}

// ChatMessage represents a simple user/assistant exchange used by the refinement chat UI.
type ChatMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt,omitempty"`
}
