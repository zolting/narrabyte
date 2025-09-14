package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"narrabyte/internal/llm/tools"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	Key             string
	fileHistoryMu   sync.Mutex
	fileOpenHistory []string
	baseRoot        string
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

	return &OpenAIClient{ChatModel: *model, Key: key}, err
}

// SetListDirectoryBaseRoot binds the list-directory tools to a specific base directory.
// Example: SetListDirectoryBaseRoot("/path/to/project") then tool input "frontend"
// resolves to "/path/to/project/frontend".
func (o *OpenAIClient) SetListDirectoryBaseRoot(root string) {
	o.baseRoot = root
	tools.SetListDirectoryBaseRoot(root)
}

func (o *OpenAIClient) ExploreCodebaseDemo(ctx context.Context, codebasePath string) (string, error) {
	// Initialize tools for this session
	allTools, err := o.InitTools()
	if err != nil {
		log.Printf("Error initializing tools: %v", err)
		return "", err
	}

	o.SetListDirectoryBaseRoot(codebasePath)

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
				"\n\nHow does app settings synchronization work? Put your explanation in a file called explanation.md at the project root."),
	}

	// Build a ReAct agent with the tool-callable model and tools config
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &o.ChatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: allTools,
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
	}

	if finalContent == "" {
		return "", fmt.Errorf("no assistant content produced during streaming")
	}

	println("OUT MESSAGE CONTENT (streamed): \n\n", finalContent)
	return finalContent, nil
}

// recordOpenedFile appends a file path to the session history if not already present.
func (o *OpenAIClient) recordOpenedFile(p string) {
	if o == nil {
		return
	}
	o.fileHistoryMu.Lock()
	defer o.fileHistoryMu.Unlock()
	norm := filepath.ToSlash(strings.TrimSpace(p))
	if norm == "" {
		return
	}
	for _, existing := range o.fileOpenHistory {
		if existing == norm {
			return
		}
	}
	o.fileOpenHistory = append(o.fileOpenHistory, norm)
}

// ResetFileOpenHistory clears the in-memory history for the current client session.
func (o *OpenAIClient) ResetFileOpenHistory() {
	if o == nil {
		return
	}
	o.fileHistoryMu.Lock()
	o.fileOpenHistory = nil
	o.fileHistoryMu.Unlock()
}

// FileOpenHistory returns a copy of the file-open history for the most recent session.
func (o *OpenAIClient) FileOpenHistory() []string {
	if o == nil {
		return nil
	}
	o.fileHistoryMu.Lock()
	defer o.fileHistoryMu.Unlock()
	out := make([]string, len(o.fileOpenHistory))
	copy(out, o.fileOpenHistory)
	return out
}

// InitTools initializes and returns all available tools for the current session.
// It resets the file-open history and wraps certain tools (e.g., read_file_tool)
// to record useful session metadata.
func (o *OpenAIClient) InitTools() ([]tool.BaseTool, error) {
	// Reset per-session file history when initializing tools
	o.ResetFileOpenHistory()

	// List directory tool
	lsDesc := tools.ToolDescription("list_directory_tool")
	if strings.TrimSpace(lsDesc) == "" {
		lsDesc = "lists the contents of a directory"
	}
	listDirectoryTool, err := utils.InferTool("list_directory_tool", lsDesc, tools.ListDirectory)
	if err != nil {
		return nil, err
	}

	// Read file tool with history capture
	readFileWithHistory := func(ctx context.Context, in *tools.ReadFileInput) (*tools.ReadFileOutput, error) {
		out, err := tools.ReadFile(ctx, in)
		if err == nil && out != nil {
			if out.Metadata == nil || out.Metadata["error"] == "" {
				o.recordOpenedFile(out.Title)
			}
		}
		return out, err
	}
	rfDesc := tools.ToolDescription("read_file_tool")
	if strings.TrimSpace(rfDesc) == "" {
		rfDesc = "reads the contents of a file"
	}
	readFileTool, err := utils.InferTool("read_file_tool", rfDesc, readFileWithHistory)
	if err != nil {
		return nil, err
	}

	// Glob tool
	globDesc := tools.ToolDescription("glob_tool")
	if strings.TrimSpace(globDesc) == "" {
		globDesc = "find files by glob pattern"
	}
	globTool, err := utils.InferTool("glob_tool", globDesc, tools.Glob)
	if err != nil {
		return nil, err
	}

	// Grep tool
	grepDesc := tools.ToolDescription("grep_tool")
	if strings.TrimSpace(grepDesc) == "" {
		grepDesc = "search file contents by regex"
	}
	grepTool, err := utils.InferTool("grep_tool", grepDesc, tools.Grep)
	if err != nil {
		return nil, err
	}

	// Write file tool with policy checks
	writeDesc := tools.ToolDescription("write_file_tool")
	if strings.TrimSpace(writeDesc) == "" {
		writeDesc = "write or create a file within the project"
	}
	writeWithPolicy := func(ctx context.Context, in *tools.WriteFileInput) (*tools.WriteFileOutput, error) {
		if in == nil {
			return nil, fmt.Errorf("input is required")
		}
		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			return nil, fmt.Errorf("file_path is required")
		}
		base := strings.TrimSpace(o.baseRoot)
		if base == "" {
			return nil, fmt.Errorf("project root not set")
		}
		// Resolve to absolute under base
		var absPath string
		if filepath.IsAbs(p) {
			absPath = p
		} else {
			absPath = filepath.Join(base, p)
		}
		absBase, err := filepath.Abs(base)
		if err != nil {
			return nil, err
		}
		absCandidate, err := filepath.Abs(absPath)
		if err != nil {
			return nil, err
		}
		relToBase, err := filepath.Rel(absBase, absCandidate)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(relToBase, "..") {
			return nil, fmt.Errorf("path escapes the configured project root")
		}
		// If file exists, enforce prior read
		if st, err := os.Stat(absCandidate); err == nil && !st.IsDir() {
			rel := filepath.ToSlash(relToBase)
			// Check client history
			o.fileHistoryMu.Lock()
			seen := false
			for _, h := range o.fileOpenHistory {
				if h == rel || h == filepath.ToSlash(absCandidate) {
					seen = true
					break
				}
			}
			o.fileHistoryMu.Unlock()
			if !seen {
				return nil, fmt.Errorf("policy violation: must read the file before writing")
			}
		}
		return tools.WriteFile(ctx, in)
	}
	writeTool, err := utils.InferTool("write_file_tool", writeDesc, writeWithPolicy)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{listDirectoryTool, readFileTool, globTool, grepTool, writeTool}, nil
}
