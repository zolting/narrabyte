package client

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/openai/openai-go/v3/responses"
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

func TestOpenAIResponsesReasoning_DefaultsToMedium(t *testing.T) {
	reasoning := openAIResponsesReasoning("")

	if reasoning == nil {
		t.Fatalf("expected reasoning config")
	}
	if reasoning.Effort != responses.ReasoningEffortMedium {
		t.Fatalf("unexpected reasoning effort: got %q want %q", reasoning.Effort, responses.ReasoningEffortMedium)
	}
}

func TestOpenAIResponsesReasoning_MapsHigh(t *testing.T) {
	reasoning := openAIResponsesReasoning("high")

	if reasoning == nil {
		t.Fatalf("expected reasoning config")
	}
	if reasoning.Effort != responses.ReasoningEffortHigh {
		t.Fatalf("unexpected reasoning effort: got %q want %q", reasoning.Effort, responses.ReasoningEffortHigh)
	}
}

func TestAgenticTextContent_ExtractsAssistantText(t *testing.T) {
	msg := &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.AssistantGenText{Text: "done"}),
		},
	}

	if got := agenticTextContent(msg); got != "done" {
		t.Fatalf("unexpected content: got %q want done", got)
	}
}

func TestAgenticReasoningContent_PreservesLeadingSpaces(t *testing.T) {
	chunks := []*schema.AgenticMessage{
		{
			Role: schema.AgenticRoleTypeAssistant,
			ContentBlocks: []*schema.ContentBlock{
				schema.NewContentBlock(&schema.Reasoning{Text: "I"}),
			},
		},
		{
			Role: schema.AgenticRoleTypeAssistant,
			ContentBlocks: []*schema.ContentBlock{
				schema.NewContentBlock(&schema.Reasoning{Text: " think"}),
			},
		},
		{
			Role: schema.AgenticRoleTypeAssistant,
			ContentBlocks: []*schema.ContentBlock{
				schema.NewContentBlock(&schema.Reasoning{Text: " we should check docs."}),
			},
		},
	}

	var reasoning strings.Builder
	for _, chunk := range chunks {
		reasoning.WriteString(agenticReasoningContent(chunk))
	}

	if got := reasoning.String(); got != "I think we should check docs." {
		t.Fatalf("unexpected reasoning content: got %q", got)
	}
}
