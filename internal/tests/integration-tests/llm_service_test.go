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
		t.Fatalf("Failed to create LLMClient: %v", err)
	}

	root, err := utils.FindProjectRoot()
	if err != nil {
		t.Fatalf("Error finding project root: %v", err)
	}

	result, err := llmClient.ExploreCodebaseDemo(testCtx, root)
	utils.NilError(t, err)
	assert.Contains(t, result, " ")
}
