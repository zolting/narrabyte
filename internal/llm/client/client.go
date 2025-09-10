package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"narrabyte/internal/llm/tools"

	"github.com/cloudwego/eino-ext/components/model/openai"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type OpenAIClient struct {
	ChatModel openai.ChatModel
	//GitToolsService tools.GitToolsService
	Key string
}

func NewOpenAIClient(ctx context.Context, key string) (*OpenAIClient, error) {
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: key,
		Model:  "gpt-5-mini",
	})

	if err != nil {
		log.Printf("Error creating OpenAI client: %v", err)
		return nil, err
	}

	return &OpenAIClient{*model, key}, err
}

// SetListDirectoryBaseRoot binds the list-directory tools to a specific base directory.
// Example: SetListDirectoryBaseRoot("/path/to/project") then tool input "frontend"
// resolves to "/path/to/project/frontend".
func (o *OpenAIClient) SetListDirectoryBaseRoot(root string) {
	tools.SetListDirectoryBaseRoot(root)
}

func (o *OpenAIClient) InvokeListDirectoryDemo(ctx context.Context, codebasePath string) (string, error) {
	// Build the list-directory tool (ls-style output)
	listDirectoryTool, err := utils.InferTool("list_directory_tool", "lists the contents of a directory", tools.ListDirectory)

	if err != nil {
		log.Printf("Error inferring tool: %v", err)
		return "", err
	}

	readFileTool, err := utils.InferTool("read_file_tool", "reads the contents of a file", tools.ReadFile)

	if err != nil {
		log.Printf("Error inferring tool: %v", err)
		return "", err
	}

	tools.SetListDirectoryBaseRoot(codebasePath)

	// Create an initial preview of the repo tree for context (textual)
	preview, err := tools.ListDirectory(ctx, &tools.ListLSInput{
		Path: codebasePath,
	})
	if err != nil {
		log.Printf("Error listing tree JSON: %v", err)
		return "", err
	}

	// Messages to drive the agent
	messages := []*schema.Message{
		schema.UserMessage(
			"Here is an initial listing of the project (capped at 100 files):\n\n" +
				preview +
				"\n\nWhat files in the project take care of synchronizing the app settings, both backend and frontend?"),
	}

	// Build a ReAct agent with the tool-callable model and tools config
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &o.ChatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{listDirectoryTool, readFileTool},
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			// Add a concise system persona before user / history
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(
				"You are a helpful codebase assistant. The user wants to understand how the codebase works. Use the tools at your disposal to answer the user's question."))
			res = append(res, input...)
			return res
		},
		MaxStep: 100,
	})

	if err != nil {
		log.Printf("Error creating react agent: %v", err)
		return "", err
	}

	// Stream the agent and log every stream event
	reader, err := agent.Stream(ctx, messages)
	if err != nil {
		log.Printf("Error starting react agent stream: %v", err)
		return "", err
	}
	if reader == nil {
		return "", fmt.Errorf("agent returned nil stream reader")
	}
	defer reader.Close()

	var finalContent string
	var lastMessage *schema.Message = nil
	for {
		msg, recvErr := reader.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				// finish
				break
			}
			// error during streaming
			log.Printf("stream recv error: %v", recvErr)
			return "", recvErr
		}

		if len(msg.ReasoningContent) > 0 {
			println("REASONING CONTENT: ", msg.ReasoningContent)
		}

		// Accumulate assistant content for the final return value
		if msg != nil && msg.Role == schema.Assistant && len(msg.Content) > 0 {
			finalContent += msg.Content
		}
		lastMessage = msg
	}

	println("COMPLETION TOKENS: ", lastMessage.ResponseMeta.Usage.CompletionTokens)
	println("PROMPT TOKENS: ", lastMessage.ResponseMeta.Usage.PromptTokens)
	println("TOTAL TOKENS: ", lastMessage.ResponseMeta.Usage.TotalTokens)

	if finalContent == "" {
		return "", fmt.Errorf("no assistant content produced during streaming")
	}

	println("OUT MESSAGE CONTENT (streamed): \n\n", finalContent)
	return finalContent, nil
}
