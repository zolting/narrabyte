package integration_tests

import (
	"context"
	"narrabyte/internal/llm/client"
	"narrabyte/internal/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Ceci est une demo simple, le context est Background() mais il faudrait peut-etre mettre un timeout plus tard pour les vraies applications
// Faudra mettre la cle a qq part aussi et pas l'exposer dans le code
func TestAddTwoNumbersTool(t *testing.T) {
	err := utils.LoadEnv()
	if err != nil {
		t.Fatalf("Error loading .env: %v", err)
	}
	testCtx := context.Background()
	apiKey := os.Getenv("OPENAI_API_KEY")
	llmClient, err := client.NewOpenAIClient(testCtx, apiKey)
	if err != nil {
		t.Fatalf("Failed to create OpenAIClient: %v", err)
	}

	result, err := llmClient.InvokeAdditionDemo(testCtx, 2, 3)
	utils.NilError(t, err)
	assert.Contains(t, result, "5")
}
