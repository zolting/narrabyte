package unit_tests

import (
	"narrabyte/internal/services"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestConvertConversationFiltersAndLabels(t *testing.T) {
	messages := []*schema.Message{
		{Role: schema.User, Content: "  initial context   "},
		nil,
		{Role: schema.Tool, Content: "tool output"},
		{Role: schema.Assistant, Content: "  reply "},
		{Role: schema.User, Content: " follow-up question "},
		{Role: schema.Assistant, Content: "   "},
	}

	converted := services.ConvertConversation(messages)
	if len(converted) != 3 {
		t.Fatalf("expected 3 messages after filtering, got %d", len(converted))
	}

	if converted[0].Role != "context" || converted[0].Content != "initial context" {
		t.Fatalf("expected first message to be context, got %#v", converted[0])
	}

	if converted[1].Role != "assistant" || converted[1].Content != "reply" {
		t.Fatalf("unexpected assistant message: %#v", converted[1])
	}

	if converted[2].Role != "user" || converted[2].Content != "follow-up question" {
		t.Fatalf("unexpected user message: %#v", converted[2])
	}
}

func TestConvertConversationEmptyInput(t *testing.T) {
	if out := services.ConvertConversation(nil); out != nil {
		t.Fatalf("expected nil input to produce nil output")
	}
}
