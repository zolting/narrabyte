package unit_tests

import (
	"github.com/cloudwego/eino/schema"
	"narrabyte/internal/llm/client"
	"testing"
)

func TestCloneDocGenerationRequestOverridesPaths(t *testing.T) {
	original := &client.DocGenerationRequest{
		ProjectID:         42,
		ProjectName:       "demo",
		DocumentationPath: "/docs/original",
		WorkspacePath:     "/docs/workspace",
		CodebasePath:      "/code/original",
		SourceBranch:      "main",
		TargetBranch:      "docs",
		SourceCommit:      "abc123",
		Diff:              "diff",
		ChangedFiles:      []string{"a.md", "b.md"},
	}

	cloned := client.CloneDocGenerationRequest(original, "/docs/override", "/code/override", "/docs/override")
	if cloned == original {
		t.Fatalf("expected clone to produce a new pointer")
	}

	if cloned.DocumentationPath != "/docs/override" {
		t.Fatalf("expected documentation path override, got %q", cloned.DocumentationPath)
	}
	if cloned.CodebasePath != "/code/override" {
		t.Fatalf("expected code path override, got %q", cloned.CodebasePath)
	}
	if cloned.WorkspacePath != "/docs/override" {
		t.Fatalf("expected workspace path override, got %q", cloned.WorkspacePath)
	}

	// Mutate clone slices to ensure deep copy semantics.
	cloned.ChangedFiles[0] = "changed.md"
	if original.ChangedFiles[0] != "a.md" {
		t.Fatalf("expected original changed files to remain untouched, got %q", original.ChangedFiles[0])
	}
}

func TestCloneDocGenerationRequestNil(t *testing.T) {
	if client.CloneDocGenerationRequest(nil, "", "", "") != nil {
		t.Fatalf("expected nil request to remain nil")
	}
}

func TestCloneMessagesDeepCopy(t *testing.T) {
	original := []*schema.Message{{
		Role:    schema.Assistant,
		Content: "assistant reply",
		ToolCalls: []schema.ToolCall{{
			ID:   "call-1",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "do",
				Arguments: "{}",
			},
		}},
		Extra: map[string]any{"key": "value"},
	}}

	cloned := client.CloneMessages(original)
	if len(cloned) != 1 {
		t.Fatalf("expected a single cloned message, got %d", len(cloned))
	}
	if cloned[0] == original[0] {
		t.Fatalf("expected cloned message pointer to differ from original")
	}

	cloned[0].Content = "mutated"
	cloned[0].ToolCalls[0].ID = "mutated"
	cloned[0].Extra["key"] = "mutated"

	if original[0].Content != "assistant reply" {
		t.Fatalf("expected original content unchanged, got %q", original[0].Content)
	}
	if original[0].ToolCalls[0].ID != "call-1" {
		t.Fatalf("expected original tool call untouched, got %q", original[0].ToolCalls[0].ID)
	}
	if original[0].Extra["key"] != "value" {
		t.Fatalf("expected original extra map untouched, got %v", original[0].Extra["key"])
	}
}

func TestOpenAIClientDocSessionIsolation(t *testing.T) {
	c := &client.OpenAIClient{}
	session := &client.DocSessionState{
		Request: &client.DocGenerationRequest{
			ProjectID:         7,
			DocumentationPath: "/docs",
			WorkspacePath:     "/docs/workspace",
			CodebasePath:      "/code",
			SourceBranch:      "feature",
			TargetBranch:      "main",
			ChangedFiles:      []string{"docs/readme.md"},
		},
		SystemPrompt: "prompt",
		Messages: []*schema.Message{{
			Role:    schema.User,
			Content: "original",
			Extra:   map[string]any{"flag": "keep"},
		}},
	}

	c.StoreDocSession(session)

	snapshot := c.DocSessionSnapshot()
	if snapshot == nil {
		t.Fatalf("expected snapshot to be available")
	}
	if snapshot == session {
		t.Fatalf("expected stored session to be cloned")
	}

	snapshot.Request.DocumentationPath = "/other"
	snapshot.Messages[0].Content = "mutated"
	snapshot.Messages[0].Extra["flag"] = "changed"

	req := c.DocSessionRequest()
	if req.DocumentationPath != "/docs" {
		t.Fatalf("expected stored documentation path to remain '/docs', got %q", req.DocumentationPath)
	}
	if req.WorkspacePath != "/docs/workspace" {
		t.Fatalf("expected stored workspace path to remain '/docs/workspace', got %q", req.WorkspacePath)
	}

	msgs := c.DocConversationMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected a single stored message, got %d", len(msgs))
	}
	if msgs[0].Content != "original" {
		t.Fatalf("expected stored message content to remain 'original', got %q", msgs[0].Content)
	}
	if msgs[0].Extra["flag"] != "keep" {
		t.Fatalf("expected stored extra map untouched, got %v", msgs[0].Extra["flag"])
	}
}

func TestStoreDocSessionClearsState(t *testing.T) {
	c := &client.OpenAIClient{}
	c.StoreDocSession(&client.DocSessionState{Request: &client.DocGenerationRequest{ProjectID: 1}})
	c.StoreDocSession(nil)
	if c.DocSessionRequest() != nil {
		t.Fatalf("expected session request to be cleared when storing nil session")
	}
	if c.DocConversationMessages() != nil {
		t.Fatalf("expected conversation to be cleared when storing nil session")
	}
}
