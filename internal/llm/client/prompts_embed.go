package client

import "embed"

// embeddedPrompts holds the built-in prompt templates so packaged executables
// can load them without needing access to the source tree.
//
//go:embed prompts/*.txt
var embeddedPrompts embed.FS
