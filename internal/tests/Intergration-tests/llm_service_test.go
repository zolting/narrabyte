package Intergration_tests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"narrabyte/internal/llm/client"
	"narrabyte/internal/tests/utils"
	"testing"
)

// Ceci est une demo simple, le context est Background() mais il faudrait peut-etre mettre un timeout plus tard pour les vraies applications
// Faudra mettre la cle a qq part aussi et pas l'exposer dans le code
func TestAddTwoNumbersTool(t *testing.T) {
	testCtx := context.Background()
	apiKey := "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"

	llmClient, err := client.NewOpenAIClient(testCtx, apiKey)
	if err != nil {
		t.Fatalf("Failed to create OpenAIClient: %v", err)
	}

	result, err := llmClient.InvokeAdditionDemo(testCtx, 2, 3)
	utils.NilError(t, err)
	assert.Contains(t, result, "5")
}
