package client

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func msg(role schema.RoleType, content string) adk.Message {
	return &schema.Message{Role: role, Content: content}
}

func TestNormalizeConversationHistory_PreservesValidHistory(t *testing.T) {
	original := []adk.Message{
		msg(schema.User, "first"),
		msg(schema.Assistant, "reply"),
	}

	result, changed := normalizeConversationHistory(original, "ignored")

	if changed {
		t.Fatalf("expected no change, got changed history")
	}
	if len(result) != len(original) {
		t.Fatalf("unexpected length: got %d want %d", len(result), len(original))
	}
	for i := range original {
		if result[i] != original[i] {
			t.Fatalf("message pointer at %d changed", i)
		}
	}
}

func TestNormalizeConversationHistory_AllowsLeadingSystem(t *testing.T) {
	original := []adk.Message{
		msg(schema.System, "sys"),
		msg(schema.User, "first"),
	}

	result, changed := normalizeConversationHistory(original, "ignored")

	if changed {
		t.Fatalf("expected no change when first non-system is user")
	}
	if len(result) != len(original) {
		t.Fatalf("unexpected length: got %d want %d", len(result), len(original))
	}
}

func TestNormalizeConversationHistory_DropsLeadingAssistant(t *testing.T) {
	original := []adk.Message{
		msg(schema.Assistant, "intro"),
		msg(schema.User, "question"),
		msg(schema.Assistant, "answer"),
	}

	result, changed := normalizeConversationHistory(original, "fallback")

	if !changed {
		t.Fatalf("expected change when leading assistant present")
	}
	if len(result) != 2 {
		t.Fatalf("unexpected length: got %d want 2", len(result))
	}
	if result[0].Role != schema.User {
		t.Fatalf("first message role %q, expected user", result[0].Role)
	}
	if result[0].Content != "question" {
		t.Fatalf("unexpected first content: %q", result[0].Content)
	}
}

func TestNormalizeConversationHistory_InsertsFallbackWhenNoUser(t *testing.T) {
	original := []adk.Message{
		msg(schema.Assistant, "reply"),
	}

	result, changed := normalizeConversationHistory(original, "fallback message")

	if !changed {
		t.Fatalf("expected change when inserting fallback")
	}
	if len(result) != 2 {
		t.Fatalf("unexpected length: got %d want 2", len(result))
	}
	if result[0].Role != schema.User {
		t.Fatalf("first message role %q, expected user", result[0].Role)
	}
	if result[0].Content != "fallback message" {
		t.Fatalf("fallback content mismatch: %q", result[0].Content)
	}
}

func TestNormalizeConversationHistory_UsesDefaultFallbackWhenEmpty(t *testing.T) {
	original := []adk.Message{
		msg(schema.Assistant, "reply"),
	}

	result, changed := normalizeConversationHistory(original, "")

	if !changed {
		t.Fatalf("expected change when inserting default fallback")
	}
	if result[0].Role != schema.User {
		t.Fatalf("first message role %q, expected user", result[0].Role)
	}
	if result[0].Content == "" {
		t.Fatalf("default fallback content should not be empty")
	}
}
