package models

// LLMModel represents a single language model option exposed to the UI.
type LLMModel struct {
	Key             string `json:"key"`
	DisplayName     string `json:"displayName"`
	APIName         string `json:"apiName"`
	ProviderID      string `json:"providerId"`
	ProviderName    string `json:"providerName"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
	Thinking        *bool  `json:"thinking,omitempty"`
	Enabled         bool   `json:"enabled"`
}

// LLMModelGroup groups models by their provider for presentation.
type LLMModelGroup struct {
	ProviderID   string     `json:"providerId"`
	ProviderName string     `json:"providerName"`
	Models       []LLMModel `json:"models"`
}
