package integration_tests

import (
	"context"
	"narrabyte/internal/llm/client"
	"narrabyte/internal/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExploreCodebase(t *testing.T) {
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

	result, err := llmClient.InvokeListDirectoryDemo(testCtx, ".")
	utils.NilError(t, err)
	assert.Contains(t, result, " ")
}
