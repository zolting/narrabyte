package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"narrabyte/internal/events"
	"narrabyte/internal/llm/tools"
	"narrabyte/internal/utils"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	einoUtils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type OpenAIClient struct {
	ChatModel       openai.ChatModel
	Key             string
	fileHistoryMu   sync.Mutex
	fileOpenHistory []string
	baseRoot        string

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
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

// watch out pour le contexte ici
func (o *OpenAIClient) StartStream(ctx context.Context) context.Context {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return ctx
	}
	o.running = true
	ctx, cancel := context.WithCancel(ctx)
	o.cancel = cancel
	o.mu.Unlock()
	return ctx
}

func (o *OpenAIClient) StopStream() {
	o.mu.Lock()
	defer o.mu.Unlock()
	cancel := o.cancel
	if cancel != nil {
		cancel()
	}
	o.running = false
	o.cancel = nil
}

// SetListDirectoryBaseRoot binds the list-directory tools to a specific base directory.
// Example: SetListDirectoryBaseRoot("/path/to/project") then tool input "frontend"
// resolves to "/path/to/project/frontend".
func (o *OpenAIClient) SetListDirectoryBaseRoot(root string) {
	// Normalize to absolute base root for consistent absolute-path semantics
	abs := root
	if r := strings.TrimSpace(root); r != "" {
		if a, err := filepath.Abs(r); err == nil {
			abs = a
		}
	}
	o.baseRoot = abs
	tools.SetListDirectoryBaseRoot(abs)
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
	allTools, err := o.initTools()
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
		"\n\nHow does the git diff frontend component work? Add your explanation by editing the explanations.md file, between App Settings and Repo Linking. You must use the edit tool to edit the file, not the write tool. Keep the current content. End by creating a file called haiku.txt in the same directory as the explanations. The haiku should be a short poem about the git diff frontend component.")

	//send events here?
	var lastMessage string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			if errors.Is(event.Err, context.Canceled) {
				return "", context.Canceled
			}
			return "", event.Err
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

	events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing complete"))
	log.Printf("Last event message: %s", lastMessage)
	return lastMessage, nil
}

// recordOpenedFile appends a file path to the session history if not already present.
func (o *OpenAIClient) recordOpenedFile(p string) {
	if o == nil {
		return
	}
	o.fileHistoryMu.Lock()
	defer o.fileHistoryMu.Unlock()
	// Normalize to absolute path; join with baseRoot when needed
	in := strings.TrimSpace(p)
	if in == "" {
		return
	}
	var abs string
	if filepath.IsAbs(in) {
		abs = in
	} else if strings.TrimSpace(o.baseRoot) != "" {
		abs = filepath.Join(o.baseRoot, in)
	} else {
		abs = in
	}
	// Best-effort Abs cleanup
	if a, err := filepath.Abs(abs); err == nil {
		abs = a
	}
	norm := filepath.ToSlash(abs)
	if slices.Contains(o.fileOpenHistory, norm) {
		return
	}
	o.fileOpenHistory = append(o.fileOpenHistory, norm)
}

// resolveAbsWithinBase resolves an input path to an absolute path under the configured base root.
// Returns the absolute candidate even when it escapes base so callers can include it in messages.
func (o *OpenAIClient) resolveAbsWithinBase(p string) (abs string, err error) {
	base := strings.TrimSpace(o.baseRoot)
	if base == "" {
		return "", fmt.Errorf("project root not set")
	}
	in := strings.TrimSpace(p)
	if in == "" {
		return "", fmt.Errorf("file_path is required")
	}
	// Build candidate absolute path
	if filepath.IsAbs(in) {
		abs = in
	} else {
		abs = filepath.Join(base, in)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return abs, err
	}
	absCandidate, err := filepath.Abs(abs)
	if err != nil {
		return absCandidate, err
	}
	relToBase, err := filepath.Rel(absBase, absCandidate)
	if err != nil {
		return absCandidate, err
	}
	if strings.HasPrefix(relToBase, "..") {
		return absCandidate, fmt.Errorf("path escapes the configured project root")
	}
	return absCandidate, nil
}

// hasRead checks if the absolute path has been read in this session.
func (o *OpenAIClient) hasRead(absPath string) bool {
	norm := filepath.ToSlash(strings.TrimSpace(absPath))
	if norm == "" {
		return false
	}
	o.fileHistoryMu.Lock()
	defer o.fileHistoryMu.Unlock()
	return slices.Contains(o.fileOpenHistory, norm)
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

// initTools initializes and returns all available tools for the current session.
// It resets the file-open history and wraps certain tools (e.g., read_file_tool)
// to record useful session metadata.
func (o *OpenAIClient) initTools() ([]tool.BaseTool, error) {
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
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("WriteFile(policy): starting"))

		if in == nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile(policy): input is required"))
			return &tools.WriteFileOutput{
				Title:  "",
				Output: "Format error: input is required",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}

		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile(policy): file_path is required"))
			return &tools.WriteFileOutput{
				Title:  "",
				Output: "Format error: file_path is required",
				Metadata: map[string]string{
					"error": "format_error",
				},
			}, nil
		}

		events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("WriteFile(policy): resolving '%s'", p)))
		absCandidate, rerr := o.resolveAbsWithinBase(p)
		if rerr != nil {
			if rerr.Error() == "project root not set" {
				events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile(policy): project root not set"))
				return &tools.WriteFileOutput{
					Title:    "",
					Output:   "Format error: project root not set",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			if strings.Contains(rerr.Error(), "escapes") {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("WriteFile(policy): path escapes the configured project root"))
				return &tools.WriteFileOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Format error: path escapes the configured project root",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile(policy): resolve error: %v", rerr)))
			return nil, rerr
		}

		// If file exists, enforce prior read
		if st, err := os.Stat(absCandidate); err == nil && !st.IsDir() {
			if !o.hasRead(absCandidate) {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("WriteFile(policy): policy violation - must read before write"))
				return &tools.WriteFileOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Policy error: must read the file before writing",
					Metadata: map[string]string{"error": "policy_violation"},
				}, nil
			}
		}

		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("WriteFile(policy): invoking underlying WriteFile for '%s'", filepath.ToSlash(absCandidate))))
		out, err := tools.WriteFile(ctx, in)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile(policy): underlying error: %v", err)))
			return out, err
		}
		title := ""
		if out != nil && strings.TrimSpace(out.Title) != "" {
			title = filepath.ToSlash(out.Title)
		}
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("WriteFile(policy): done%s", func() string {
			if title == "" {
				return ""
			}
			return " for '" + title + "'"
		}())))
		return out, nil
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
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("EditFile(policy): starting"))

		// Validate input
		if in == nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("EditFile(policy): input is required"))
			return &tools.EditOutput{
				Title:    "",
				Output:   "Format error: input is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			events.Emit(ctx, events.LLMEventTool, events.NewError("EditFile(policy): file_path is required"))
			return &tools.EditOutput{
				Title:    "",
				Output:   "Format error: file_path is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}

		// Resolve absolute path and ensure it is under base
		events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("EditFile(policy): resolving '%s'", p)))
		absCandidate, rerr := o.resolveAbsWithinBase(p)
		if rerr != nil {
			if rerr.Error() == "project root not set" {
				events.Emit(ctx, events.LLMEventTool, events.NewError("EditFile(policy): project root not set"))
				return &tools.EditOutput{
					Title:    "",
					Output:   "Format error: project root not set",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			if strings.Contains(rerr.Error(), "escapes") {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("EditFile(policy): path escapes the configured project root"))
				return &tools.EditOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Format error: path escapes the configured project root",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("EditFile(policy): resolve error: %v", rerr)))
			return nil, rerr
		}

		// If file exists, enforce prior read
		if st, err := os.Stat(absCandidate); err == nil && !st.IsDir() {
			if !o.hasRead(absCandidate) {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("EditFile(policy): policy violation - must read before edit"))
				return &tools.EditOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Policy error: must read the file before editing",
					Metadata: map[string]string{"error": "policy_violation"},
				}, nil
			}
		}

		// Invoke underlying Edit tool
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("EditFile(policy): invoking underlying Edit for '%s'", filepath.ToSlash(absCandidate))))
		out, err := tools.Edit(ctx, in)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("EditFile(policy): underlying error: %v", err)))
			return out, err
		}

		title := ""
		if out != nil && strings.TrimSpace(out.Title) != "" {
			title = filepath.ToSlash(out.Title)
		}
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(func() string {
			if title == "" {
				return "EditFile(policy): done"
			}
			return "EditFile(policy): done for '" + title + "'"
		}()))
		return out, nil
	}

	editTool, err := einoUtils.InferTool("edit_tool", editDesc, editWithPolicy)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{listDirectoryTool, readFileTool, globTool, grepTool, writeTool, editTool}, nil
}
