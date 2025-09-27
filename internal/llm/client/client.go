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
	"github.com/cloudwego/eino/schema"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type OpenAIClient struct {
	ChatModel       openai.ChatModel
	Key             string
	fileHistoryMu   sync.Mutex
	fileOpenHistory []string
	baseRoot        string
	docRoot         string
	codeRoot        string
	sourceBranch    string
	targetBranch    string
	sourceCommit    string
	targetCommit    string
	codeSnapshot    *tools.GitSnapshot

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc

	docSessionMu    sync.Mutex
	docSessionState *DocSessionState
}

type DocGenerationRequest struct {
	ProjectID         uint
	ProjectName       string
	CodebasePath      string
	DocumentationPath string
	WorkspacePath     string
	SourceBranch      string
	TargetBranch      string
	SourceCommit      string
	Diff              string
	ChangedFiles      []string
}

type DocGenerationResponse struct {
	Summary  string
	Messages []*schema.Message
}

type DocSessionState struct {
	Request      *DocGenerationRequest
	SystemPrompt string
	Messages     []*schema.Message
}

func (s *DocSessionState) clone() *DocSessionState {
	if s == nil {
		return nil
	}
	return &DocSessionState{
		Request:      CloneDocGenerationRequest(s.Request, "", "", ""),
		SystemPrompt: s.SystemPrompt,
		Messages:     CloneMessages(s.Messages),
	}
}

func CloneDocGenerationRequest(req *DocGenerationRequest, docRoot, codeRoot, workspaceRoot string) *DocGenerationRequest {
	if req == nil {
		return nil
	}
	copyReq := *req
	if docRoot != "" {
		copyReq.DocumentationPath = docRoot
	}
	if codeRoot != "" {
		copyReq.CodebasePath = codeRoot
	}
	if workspaceRoot != "" {
		copyReq.WorkspacePath = workspaceRoot
	}
	if len(req.ChangedFiles) > 0 {
		copyReq.ChangedFiles = append([]string(nil), req.ChangedFiles...)
	} else {
		copyReq.ChangedFiles = nil
	}
	return &copyReq
}

func cloneMessage(msg *schema.Message) *schema.Message {
	if msg == nil {
		return nil
	}
	cloned := &schema.Message{
		Role:             msg.Role,
		Content:          msg.Content,
		Name:             msg.Name,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		ReasoningContent: msg.ReasoningContent,
	}
	if len(msg.MultiContent) > 0 {
		cloned.MultiContent = make([]schema.ChatMessagePart, len(msg.MultiContent))
		copy(cloned.MultiContent, msg.MultiContent)
	}
	if len(msg.ToolCalls) > 0 {
		cloned.ToolCalls = make([]schema.ToolCall, len(msg.ToolCalls))
		copy(cloned.ToolCalls, msg.ToolCalls)
	}
	if msg.Extra != nil {
		extra := make(map[string]any, len(msg.Extra))
		for k, v := range msg.Extra {
			extra[k] = v
		}
		cloned.Extra = extra
	}
	return cloned
}

func CloneMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		cloned = append(cloned, cloneMessage(msg))
	}
	return cloned
}

func (o *OpenAIClient) StoreDocSession(session *DocSessionState) {
	if o == nil {
		return
	}
	o.docSessionMu.Lock()
	defer o.docSessionMu.Unlock()
	if session == nil {
		o.docSessionState = nil
		return
	}
	o.docSessionState = session.clone()
}

func (o *OpenAIClient) DocSessionSnapshot() *DocSessionState {
	if o == nil {
		return nil
	}
	o.docSessionMu.Lock()
	defer o.docSessionMu.Unlock()
	if o.docSessionState == nil {
		return nil
	}
	return o.docSessionState.clone()
}

func (o *OpenAIClient) DocSessionRequest() *DocGenerationRequest {
	snapshot := o.DocSessionSnapshot()
	if snapshot == nil {
		return nil
	}
	return CloneDocGenerationRequest(snapshot.Request, "", "", "")
}

func (o *OpenAIClient) DocConversationMessages() []*schema.Message {
	snapshot := o.DocSessionSnapshot()
	if snapshot == nil {
		return nil
	}
	return CloneMessages(snapshot.Messages)
}

func NewOpenAIClient(ctx context.Context, key string) (*OpenAIClient, error) {
	// temperature := float32(0)
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: key,
		Model:  "gpt-5-mini",
		// APIKey:  "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",
		// Model:   "x-ai/grok-4-fast:free",
		// BaseURL: "https://openrouter.ai/api/v1",
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

// IsRunning reports whether a session is currently active.
func (o *OpenAIClient) IsRunning() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.running
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
	tools.SetGitSnapshot(nil)
	tools.SetListDirectoryBaseRoot(abs)
}

// loadSystemPrompt loads the system instruction from the demo.txt file
func (o *OpenAIClient) loadPrompt(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("prompt name is required")
	}
	projectRoot, err := utils.FindProjectRoot()
	if err != nil {
		return "", err
	}
	promptPath := filepath.Join(projectRoot, "internal", "llm", "client", "prompts", name)
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (o *OpenAIClient) ExploreCodebaseDemo(ctx context.Context, codebasePath string) (string, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("ExploreCodebaseDemo: starting"))

	// Initialize tools
	allTools, err := o.initTools()
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ExploreCodebaseDemo: init tools error: %v", err)))
		return "", err
	}

	o.SetListDirectoryBaseRoot(codebasePath)

	// Load system prompt
	systemPrompt, err := o.loadPrompt("demo.txt")
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ExploreCodebaseDemo: load system prompt error: %v", err)))
		return "", err
	}
	events.Emit(ctx, events.LLMEventTool, events.NewDebug("ExploreCodebaseDemo: system prompt loaded"))

	// Repo preview
	preview, err := tools.ListDirectory(ctx, &tools.ListLSInput{Path: codebasePath})
	if err != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ExploreCodebaseDemo: preview error: %v", err)))
		return "", err
	}
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("ExploreCodebaseDemo: repository preview generated"))

	// Build agent
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
		events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ExploreCodebaseDemo: agent creation error: %v", err)))
		return "", err
	}
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("ExploreCodebaseDemo: agent created"))

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	iter := runner.Query(ctx, "Here is an initial listing of the project (capped at 100 files):\n\n"+
		preview.Output+
		"\n\nHow does the git diff frontend component work? Add your explanation by editing the explanations.md file, between App Settings and Repo Linking. You must use the edit tool to edit the file, not the write tool. Keep the current content. End by creating a file called haiku.txt in the same directory as the explanations. The haiku should be a short poem about the git diff frontend component.")

	var lastMessage string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if errors.Is(event.Err, context.Canceled) {
				events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing canceled"))
				return "", context.Canceled
			}
			events.Emit(ctx, events.LLMEventDone, events.NewError("LLM processing error"))
			return "", event.Err
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			events.Emit(ctx, events.LLMEventDone, events.NewError(fmt.Sprintf("LLM message error: %v", err)))
			continue
		}
		lastMessage = msg.Content
	}

	events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing complete"))

	return lastMessage, nil
}

func (o *OpenAIClient) GenerateDocs(ctx context.Context, req *DocGenerationRequest) (*DocGenerationResponse, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("GenerateDocs: initializing"))
	if req == nil {
		return nil, fmt.Errorf("Request is required")
	}
	workspaceRoot := strings.TrimSpace(req.WorkspacePath)
	if workspaceRoot == "" {
		workspaceRoot = strings.TrimSpace(req.DocumentationPath)
	}
	if workspaceRoot == "" {
		return nil, fmt.Errorf("documentation workspace path is required")
	}
	if strings.TrimSpace(req.CodebasePath) == "" {
		return nil, fmt.Errorf("codebase path is required")
	}
	workspaceAbs, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, err
	}
	docRootRaw := strings.TrimSpace(req.DocumentationPath)
	if docRootRaw == "" {
		docRootRaw = workspaceAbs
	}
	docRoot, err := filepath.Abs(docRootRaw)
	if err != nil {
		return nil, err
	}
	codeRoot, err := filepath.Abs(req.CodebasePath)
	if err != nil {
		return nil, err
	}
	o.docRoot = workspaceAbs
	o.codeRoot = codeRoot
	o.sourceBranch = strings.TrimSpace(req.SourceBranch)
	o.targetBranch = strings.TrimSpace(req.TargetBranch)
	o.sourceCommit = strings.TrimSpace(req.SourceCommit)
	o.targetCommit = ""
	o.SetListDirectoryBaseRoot(workspaceAbs)

	if err := o.prepareSnapshots(ctx); err != nil {
		return nil, err
	}

	docListing, err := o.captureListing(ctx, workspaceAbs)
	if err != nil {
		return nil, fmt.Errorf("failed to list documentation root: %w", err)
	}
	codeListing, err := o.captureListing(ctx, codeRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list codebase root: %w", err)
	}

	toolsForSession, err := o.initDocumentationTools(workspaceAbs, codeRoot)
	if err != nil {
		return nil, err
	}

	systemPrompt, err := o.loadPrompt("generate_docs.txt")
	if err != nil {
		return nil, err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: &o.ChatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolsForSession,
			},
		},
		Name:        "Documentation Assistant",
		Description: "Analyzes code diffs and proposes documentation updates",
		Instruction: systemPrompt,
	})
	if err != nil {
		return nil, err
	}

	changedList := "(none)"
	if len(req.ChangedFiles) > 0 {
		var b strings.Builder
		for _, f := range req.ChangedFiles {
			b.WriteString("- ")
			b.WriteString(filepath.ToSlash(f))
			b.WriteString("\n")
		}
		changedList = strings.TrimSpace(b.String())
	}

	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("Project: %s\n", strings.TrimSpace(req.ProjectName)))
	promptBuilder.WriteString(fmt.Sprintf("Source branch: %s\n", strings.TrimSpace(req.SourceBranch)))
	if commit := strings.TrimSpace(req.SourceCommit); commit != "" {
		promptBuilder.WriteString(fmt.Sprintf("Source branch commit: %s\n", commit))
	}
	promptBuilder.WriteString(fmt.Sprintf("Target branch: %s\n\n", strings.TrimSpace(req.TargetBranch)))
	promptBuilder.WriteString("Documentation repository root: \n")
	promptBuilder.WriteString(filepath.ToSlash(workspaceAbs))
	promptBuilder.WriteString("\n\n")
	promptBuilder.WriteString("Codebase repository root: \n")
	promptBuilder.WriteString(filepath.ToSlash(codeRoot))
	promptBuilder.WriteString("\n\n")
	promptBuilder.WriteString("Changed source files (relative to codebase root):\n")
	promptBuilder.WriteString(changedList)
	promptBuilder.WriteString("\n\n")
	promptBuilder.WriteString("<documentation_repo_listing>\n")
	promptBuilder.WriteString(docListing)
	promptBuilder.WriteString("\n</documentation_repo_listing>\n\n")
	promptBuilder.WriteString("<codebase_repo_listing>\n")
	promptBuilder.WriteString(codeListing)
	promptBuilder.WriteString("\n</codebase_repo_listing>\n\n")
	promptBuilder.WriteString("<git_diff>\n")
	promptBuilder.WriteString(req.Diff)
	promptBuilder.WriteString("\n</git_diff>")

	initialMessages := []*schema.Message{schema.UserMessage(promptBuilder.String())}
	session := &DocSessionState{
		Request:      CloneDocGenerationRequest(req, docRoot, codeRoot, docRoot),
		SystemPrompt: systemPrompt,
		Messages:     CloneMessages(initialMessages),
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	iter := runner.Run(ctx, initialMessages)
	var lastMessage string
	conversation := CloneMessages(initialMessages)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if errors.Is(event.Err, context.Canceled) {
				events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing canceled"))
				return nil, context.Canceled
			}
			events.Emit(ctx, events.LLMEventDone, events.NewError("LLM processing error"))
			return nil, event.Err
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("GenerateDocs: message error: %v", err)))
			continue
		}
		cloned := cloneMessage(msg)
		if cloned != nil {
			conversation = append(conversation, cloned)
			lastMessage = cloned.Content
		}
	}

	events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing complete"))
	session.Messages = conversation
	o.StoreDocSession(session)
	return &DocGenerationResponse{
		Summary:  strings.TrimSpace(lastMessage),
		Messages: CloneMessages(conversation),
	}, nil
}

func (o *OpenAIClient) ApplyDocFeedback(ctx context.Context, feedback string) (*DocGenerationResponse, error) {
	feedback = strings.TrimSpace(feedback)
	if feedback == "" {
		return nil, fmt.Errorf("feedback is required")
	}

	session := o.DocSessionSnapshot()
	if session == nil || session.Request == nil {
		return nil, fmt.Errorf("no documentation session available to continue")
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("ApplyDocFeedback: continuing documentation session"))

	docRoot := strings.TrimSpace(session.Request.DocumentationPath)
	codeRoot := strings.TrimSpace(session.Request.CodebasePath)
	workspaceRoot := strings.TrimSpace(session.Request.WorkspacePath)
	if workspaceRoot == "" {
		workspaceRoot = docRoot
	}
	if docRoot == "" || codeRoot == "" || workspaceRoot == "" {
		return nil, fmt.Errorf("documentation session paths are not available")
	}

	docRootAbs, err := filepath.Abs(docRoot)
	if err != nil {
		return nil, err
	}
	workspaceAbs, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, err
	}
	codeRootAbs, err := filepath.Abs(codeRoot)
	if err != nil {
		return nil, err
	}

	toolsForSession, err := o.initDocumentationTools(workspaceAbs, codeRootAbs)
	if err != nil {
		return nil, err
	}

	systemPrompt := session.SystemPrompt
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt, err = o.loadPrompt("generate_docs.txt")
		if err != nil {
			return nil, err
		}
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: &o.ChatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolsForSession,
			},
		},
		Name:        "Documentation Assistant",
		Description: "Analyzes code diffs and proposes documentation updates",
		Instruction: systemPrompt,
	})
	if err != nil {
		return nil, err
	}

	history := CloneMessages(session.Messages)
	if len(history) == 0 {
		history = []*schema.Message{}
	}
	history = append(history, schema.UserMessage(feedback))

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	iter := runner.Run(ctx, history)
	conversation := CloneMessages(history)
	var lastMessage string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if errors.Is(event.Err, context.Canceled) {
				events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing canceled"))
				return nil, context.Canceled
			}
			events.Emit(ctx, events.LLMEventDone, events.NewError("LLM processing error"))
			return nil, event.Err
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ApplyDocFeedback: message error: %v", err)))
			continue
		}
		cloned := cloneMessage(msg)
		if cloned != nil {
			conversation = append(conversation, cloned)
			lastMessage = cloned.Content
		}
	}

	events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing complete"))

	updatedSession := &DocSessionState{
		Request:      CloneDocGenerationRequest(session.Request, docRootAbs, codeRootAbs, workspaceAbs),
		SystemPrompt: systemPrompt,
		Messages:     conversation,
	}
	o.StoreDocSession(updatedSession)

	return &DocGenerationResponse{
		Summary:  strings.TrimSpace(lastMessage),
		Messages: CloneMessages(conversation),
	}, nil
}

func (o *OpenAIClient) prepareSnapshots(ctx context.Context) error {
	codeRoot := strings.TrimSpace(o.codeRoot)
	sourceCommit := strings.TrimSpace(o.sourceCommit)
	sourceBranch := strings.TrimSpace(o.sourceBranch)
	if codeRoot != "" && sourceCommit != "" {
		repo, err := git.PlainOpen(codeRoot)
		if err != nil {
			return fmt.Errorf("failed to open code repository for snapshot: %w", err)
		}
		commit, err := repo.CommitObject(plumbing.NewHash(sourceCommit))
		if err != nil {
			return fmt.Errorf("failed to resolve source commit '%s': %w", sourceCommit, err)
		}
		snapshot, err := tools.NewGitSnapshot(repo, commit, codeRoot, sourceBranch)
		if err != nil {
			return fmt.Errorf("failed to build code repository snapshot: %w", err)
		}
		o.codeSnapshot = snapshot
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf(
			"Snapshots: configured code snapshot for branch '%s' at commit %s",
			sourceBranch, sourceCommit,
		)))
	} else {
		o.codeSnapshot = nil
	}

	events.Emit(ctx, events.LLMEventTool, events.NewInfo("Snapshots: documentation tools will use the live workspace"))

	return nil
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
// Returns the absolute candidate even when it escapes base so callers can include it in Messages.
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

func (o *OpenAIClient) captureListing(ctx context.Context, root string) (string, error) {
	var listing string
	snapshot := o.snapshotForRoot(root)
	events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("CaptureListing: . [%s]", snapshotInfo(snapshot))))
	err := o.withBaseRoot(root, snapshot, func() error {
		out, err := tools.ListDirectory(ctx, &tools.ListLSInput{Path: "."})
		if err != nil {
			return err
		}
		listing = out.Output
		return nil
	})
	if err != nil {
		return "", err
	}
	return listing, nil
}

func (o *OpenAIClient) snapshotForRoot(root string) *tools.GitSnapshot {
	if pathsEqual(root, o.codeRoot) {
		return o.codeSnapshot
	}
	return nil
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

func (o *OpenAIClient) initDocumentationTools(docRoot, codeRoot string) ([]tool.BaseTool, error) {
	o.ResetFileOpenHistory()
	o.docRoot = docRoot
	o.codeRoot = codeRoot
	o.SetListDirectoryBaseRoot(docRoot)

	listDesc := tools.ToolDescription("list_directory_tool")
	if strings.TrimSpace(listDesc) == "" {
		listDesc = "lists the contents of a directory"
	}
	listWithPolicy := func(ctx context.Context, in *tools.ListLSInput) (string, error) {
		requested := "."
		var ignore []string
		if in != nil {
			if strings.TrimSpace(in.Path) != "" {
				requested = strings.TrimSpace(in.Path)
			}
			ignore = in.Ignore
		}
		root, rel, _, err := o.resolveToolPath(requested, true)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory(policy): %v", err)))
			return "", err
		}
		var output string
		snapshot := o.snapshotForRoot(root)
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ListDirectory: %s [%s]", rel, snapshotInfo(snapshot))))
		err = o.withBaseRoot(root, snapshot, func() error {
			res, innerErr := tools.ListDirectory(ctx, &tools.ListLSInput{Path: rel, Ignore: ignore})
			if innerErr != nil {
				return innerErr
			}
			output = res.Output
			return nil
		})
		return output, err
	}
	listTool, err := einoUtils.InferTool("list_directory_tool", listDesc, listWithPolicy)
	if err != nil {
		return nil, err
	}

	readDesc := tools.ToolDescription("read_file_tool")
	if strings.TrimSpace(readDesc) == "" {
		readDesc = "reads the contents of a file"
	}
	readWithPolicy := func(ctx context.Context, in *tools.ReadFileInput) (*tools.ReadFileOutput, error) {
		if in == nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("ReadFile(policy): input is required"))
			return &tools.ReadFileOutput{
				Output:   "Format error: input is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		root, rel, abs, err := o.resolveToolPath(in.FilePath, true)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile(policy): %v", err)))
			return &tools.ReadFileOutput{
				Title:    "",
				Output:   fmt.Sprintf("Policy error: %v", err),
				Metadata: map[string]string{"error": "policy_violation"},
			}, nil
		}
		var out *tools.ReadFileOutput
		snapshot := o.snapshotForRoot(root)
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("ReadFile: %s [%s]", rel, snapshotInfo(snapshot))))
		err = o.withBaseRoot(root, snapshot, func() error {
			res, innerErr := tools.ReadFile(ctx, &tools.ReadFileInput{
				FilePath: abs,
				Offset:   in.Offset,
				Limit:    in.Limit,
			})
			if innerErr != nil {
				return innerErr
			}
			out = res
			return nil
		})
		if err != nil {
			return out, err
		}
		if out != nil && (out.Metadata == nil || out.Metadata["error"] == "") {
			o.recordOpenedFile(abs)
		}
		return out, nil
	}
	readTool, err := einoUtils.InferTool("read_file_tool", readDesc, readWithPolicy)
	if err != nil {
		return nil, err
	}

	writeDesc := tools.ToolDescription("write_file_tool")
	if strings.TrimSpace(writeDesc) == "" {
		writeDesc = "write or create a file within the documentation repository"
	}
	writeWithPolicy := func(ctx context.Context, in *tools.WriteFileInput) (*tools.WriteFileOutput, error) {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("WriteFile(policy): starting"))
		if in == nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile(policy): input is required"))
			return &tools.WriteFileOutput{
				Output:   "Format error: input is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile(policy): file_path is required"))
			return &tools.WriteFileOutput{
				Output:   "Format error: file_path is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("WriteFile(policy): resolving '%s'", p)))
		absCandidate, rerr := o.resolveAbsWithinBase(p)
		if rerr != nil {
			if rerr.Error() == "project root not set" {
				events.Emit(ctx, events.LLMEventTool, events.NewError("WriteFile(policy): documentation root not set"))
				return &tools.WriteFileOutput{
					Output:   "Format error: project root not set",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			if strings.Contains(rerr.Error(), "escapes") {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("WriteFile(policy): path escapes the documentation root"))
				return &tools.WriteFileOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Format error: path escapes the documentation root",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("WriteFile(policy): resolve error: %v", rerr)))
			return nil, rerr
		}
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
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(func() string {
			if title == "" {
				return "WriteFile(policy): done"
			}
			return "WriteFile(policy): done for '" + title + "'"
		}()))
		return out, nil
	}
	writeTool, err := einoUtils.InferTool("write_file_tool", writeDesc, writeWithPolicy)
	if err != nil {
		return nil, err
	}

	editDesc := tools.ToolDescription("edit_file_tool")
	if strings.TrimSpace(editDesc) == "" {
		editDesc = "edit a file using context-aware string replacement"
	}
	editWithPolicy := func(ctx context.Context, in *tools.EditInput) (*tools.EditOutput, error) {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("EditFile(policy): starting"))
		if in == nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("EditFile(policy): input is required"))
			return &tools.EditOutput{
				Output:   "Format error: input is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			events.Emit(ctx, events.LLMEventTool, events.NewError("EditFile(policy): file_path is required"))
			return &tools.EditOutput{
				Output:   "Format error: file_path is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		events.Emit(ctx, events.LLMEventTool, events.NewDebug(fmt.Sprintf("EditFile(policy): resolving '%s'", p)))
		absCandidate, rerr := o.resolveAbsWithinBase(p)
		if rerr != nil {
			if rerr.Error() == "project root not set" {
				events.Emit(ctx, events.LLMEventTool, events.NewError("EditFile(policy): documentation root not set"))
				return &tools.EditOutput{
					Output:   "Format error: project root not set",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			if strings.Contains(rerr.Error(), "escapes") {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("EditFile(policy): path escapes the documentation root"))
				return &tools.EditOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Format error: path escapes the documentation root",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("EditFile(policy): resolve error: %v", rerr)))
			return nil, rerr
		}
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

	return []tool.BaseTool{listTool, readTool, writeTool, editTool}, nil
}

func (o *OpenAIClient) withBaseRoot(root string, snapshot *tools.GitSnapshot, fn func() error) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("base root not set")
	}
	prevRoot := o.baseRoot
	prevSnapshot := tools.CurrentGitSnapshot()
	tools.SetListDirectoryBaseRoot(root)
	tools.SetGitSnapshot(snapshot)
	o.baseRoot = root
	defer func() {
		tools.SetListDirectoryBaseRoot(prevRoot)
		tools.SetGitSnapshot(prevSnapshot)
		o.baseRoot = prevRoot
	}()
	return fn()
}

func (o *OpenAIClient) resolveToolPath(input string, allowCode bool) (root string, rel string, abs string, err error) {
	in := strings.TrimSpace(input)
	if in == "" || in == "." {
		return o.docRoot, ".", o.docRoot, nil
	}
	if filepath.IsAbs(in) {
		absInput, err := filepath.Abs(in)
		if err != nil {
			return "", "", "", err
		}
		if rel, full, ok := pathWithin(o.docRoot, absInput); ok {
			return o.docRoot, relOrDot(rel), full, nil
		}
		if allowCode {
			if rel, full, ok := pathWithin(o.codeRoot, absInput); ok {
				return o.codeRoot, relOrDot(rel), full, nil
			}
		}
		return "", "", "", fmt.Errorf("path '%s' is not within the allowed repositories", input)
	}
	candidate := filepath.Join(o.docRoot, in)
	if rel, full, ok := pathWithin(o.docRoot, candidate); ok {
		return o.docRoot, relOrDot(rel), full, nil
	}
	return "", "", "", fmt.Errorf("path '%s' escapes the documentation repository", input)
}

func pathWithin(base, candidate string) (rel string, abs string, ok bool) {
	if strings.TrimSpace(base) == "" {
		return "", "", false
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", "", false
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", "", false
	}
	relative, err := filepath.Rel(absBase, absCandidate)
	if err != nil {
		return "", "", false
	}
	if strings.HasPrefix(relative, "..") {
		return "", "", false
	}
	return relative, absCandidate, true
}

func relOrDot(rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "."
	}
	return rel
}

func snapshotInfo(snapshot *tools.GitSnapshot) string {
	if snapshot == nil {
		return "no-snapshot"
	}
	branch := snapshot.Branch()
	commit := snapshot.CommitHash().String()
	if len(commit) > 8 {
		commit = commit[:8]
	}
	if branch != "" {
		return fmt.Sprintf("%s@%s", branch, commit)
	}
	return commit
}

func pathsEqual(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	if errA != nil {
		absA = filepath.Clean(a)
	}
	absB, errB := filepath.Abs(b)
	if errB != nil {
		absB = filepath.Clean(b)
	}
	return absA == absB
}

func imageTypeByExt(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".jpg", ".jpeg":
		return "JPEG"
	case ".png":
		return "PNG"
	case ".gif":
		return "GIF"
	case ".bmp":
		return "BMP"
	case ".webp":
		return "WebP"
	default:
		return ""
	}
}
