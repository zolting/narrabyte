package assets

import _ "embed"

// ModelsData holds the raw JSON configuration for LLM models.
//
//go:embed models.json
var ModelsData []byte
