package client

import (
	"context"
	"log"
	"narrabyte/internal/llm/tools"
	"narrabyte/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	einoUtils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
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

// loadSystemPrompt loads the system instruction from the demo.txt file
func (o *OpenAIClient) loadSystemPrompt() (string, error) {
	// Get the project root by finding go.mod
	projectRoot, err := utils.FindProjectRoot()
	if err != nil {
		return "", err
	}

	promptPath := filepath.Join(projectRoot, "internal", "llm", "client", "prompts", "demo.txt")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (o *OpenAIClient) ExploreCodebaseDemo(ctx context.Context, codebasePath string) (string, error) {
	// Initialize tools for this session
	allTools, err := o.InitTools()
	if err != nil {
		log.Printf("Error initializing tools: %v", err)
		return "", err
	}

	o.SetListDirectoryBaseRoot(codebasePath)

	// Load system prompt from demo.txt
	systemPrompt, err := o.loadSystemPrompt()
	if err != nil {
		log.Printf("Error loading system prompt: %v", err)
		return "", err
	}

	// Create an initial preview of the repo tree for context (textual)
	preview, err := tools.ListDirectory(ctx, &tools.ListLSInput{
		Path: codebasePath,
	})
	if err != nil {
		log.Printf("Error listing tree JSON: %v", err)
		return "", err
	}

	// Build a ReAct agent with the tool-callable model and tools config
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: &o.ChatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
		Name:        "Codebase Assistant",
		Description: "An agent that helps the user understand a codebase.",
		Instruction: systemPrompt,
	})

	if err != nil {
		log.Printf("Error creating react agent: %v", err)
		return "", err
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	iter := runner.Query(ctx, "Here is an initial listing of the project (capped at 100 files):\n\n"+
		preview+
		"\n\nHow does the git diff frontend component work? Add your explanation by editing the file called explanations.md at the project root, between App Settings and Repo Linking. You must use the edit tool to edit the file, not the write tool. Keep the current content.")

	var lastMessage string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			log.Fatal(err)
		}
		lastMessage = msg.Content
	}

	log.Printf("Last event message: %s", lastMessage)
	return "", nil
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
	listDirectoryTool, err := einoUtils.InferTool("list_directory_tool", lsDesc, tools.ListDirectory)
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
	readFileTool, err := einoUtils.InferTool("read_file_tool", rfDesc, readFileWithHistory)
	if err != nil {
		return nil, err
	}

	// Glob tool
	globDesc := tools.ToolDescription("glob_tool")
	if strings.TrimSpace(globDesc) == "" {
		globDesc = "find files by glob pattern"
	}
	globTool, err := einoUtils.InferTool("glob_tool", globDesc, tools.Glob)
	if err != nil {
		return nil, err
	}

	// Grep tool
	grepDesc := tools.ToolDescription("grep_tool")
	if strings.TrimSpace(grepDesc) == "" {
		grepDesc = "search file contents by regex"
	}
	grepTool, err := einoUtils.InferTool("grep_tool", grepDesc, tools.Grep)
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
			return &tools.WriteFileOutput{
				Title:    "",
				Output:   "Format error: input is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			return &tools.WriteFileOutput{
				Title:    "",
				Output:   "Format error: file_path is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		base := strings.TrimSpace(o.baseRoot)
		if base == "" {
			return &tools.WriteFileOutput{
				Title:    "",
				Output:   "Format error: project root not set",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
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
			return &tools.WriteFileOutput{
				Title:    filepath.ToSlash(p),
				Output:   "Format error: path escapes the configured project root",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
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
				return &tools.WriteFileOutput{
					Title:    rel,
					Output:   "Policy error: must read the file before writing",
					Metadata: map[string]string{"error": "policy_violation"},
				}, nil
			}
		}
		return tools.WriteFile(ctx, in)
	}
	writeTool, err := einoUtils.InferTool("write_file_tool", writeDesc, writeWithPolicy)
	if err != nil {
		return nil, err
	}

	// Edit tool with policy checks
	editDesc := tools.ToolDescription("edit_file_tool")
	if strings.TrimSpace(editDesc) == "" {
		editDesc = "edit a file using context-aware string replacement"
	}
	editWithPolicy := func(ctx context.Context, in *tools.EditInput) (*tools.EditOutput, error) {
		if in == nil {
			return &tools.EditOutput{
				Title:    "",
				Output:   "Format error: input is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			return &tools.EditOutput{
				Title:    "",
				Output:   "Format error: file_path is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		base := strings.TrimSpace(o.baseRoot)
		if base == "" {
			return &tools.EditOutput{
				Title:    "",
				Output:   "Format error: project root not set",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
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
			return &tools.EditOutput{
				Title:    filepath.ToSlash(p),
				Output:   "Format error: path escapes the configured project root",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
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
				return &tools.EditOutput{
					Title:    rel,
					Output:   "Policy error: must read the file before editing",
					Metadata: map[string]string{"error": "policy_violation"},
				}, nil
			}
		}
		return tools.Edit(ctx, in)
	}
	editTool, err := einoUtils.InferTool("edit_tool", editDesc, editWithPolicy)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{listDirectoryTool, readFileTool, globTool, grepTool, writeTool, editTool}, nil
}
