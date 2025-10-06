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

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"google.golang.org/genai"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	einoUtils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const llmInstructionsNamePrefix = "llm_instructions"

type LLMClient struct {
	chatModel       model.ToolCallingChatModel
	Key             string
	fileHistoryMu   sync.Mutex
	fileOpenHistory []string
	baseRoot        string
	docRoot         string
	codeRoot        string
	docRelative     string
	sourceBranch    string
	targetBranch    string
	sourceCommit    string
	targetCommit    string
	codeSnapshot    *tools.GitSnapshot

	mu                    sync.Mutex
	running               bool
	cancel                context.CancelFunc
	conversationHistoryMu sync.Mutex
	conversationHistory   []adk.Message // Store conversation for context in refinement
}

type DocGenerationRequest struct {
	ProjectName          string
	CodebasePath         string
	DocumentationPath    string
	DocumentationRelPath string
	SourceBranch         string
	TargetBranch         string
	SourceCommit         string
	Diff                 string
	ChangedFiles         []string
	SpecificInstr        string
}

type DocRefineRequest struct {
	ProjectName          string
	CodebasePath         string
	DocumentationPath    string
	DocumentationRelPath string
	SourceBranch         string
	Instruction          string
}

type DocGenerationResponse struct {
	Summary string
}

func NewOpenAIClient(ctx context.Context, key string) (*LLMClient, error) {
	// temperature := float32(0)
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:          key,
		Model:           "gpt-5-mini",
		ReasoningEffort: openai.ReasoningEffortLevelMedium,
	})

	if err != nil {
		log.Printf("Error creating OpenAI client: %v", err)
		return nil, err
	}

	return &LLMClient{chatModel: model, Key: key}, err
}

func NewClaudeClient(ctx context.Context, key string) (*LLMClient, error) {
	model, err := claude.NewChatModel(ctx, &claude.Config{
		APIKey:    key,
		Model:     "claude-sonnet-4-5",
		MaxTokens: 4096,
	})

	if err != nil {
		log.Printf("Error creating Claude client: %v", err)
		return nil, err
	}

	return &LLMClient{chatModel: model, Key: key}, err
}

func NewGeminiClient(ctx context.Context, key string) (*LLMClient, error) {
	genai, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: key,
	})

	if err != nil {
		log.Printf("Error creating Gemini client: %v", err)
		return nil, err
	}

	model, err := gemini.NewChatModel(ctx, &gemini.Config{
		Client: genai,
		Model:  "gemini-flash-latest",
	})

	if err != nil {
		log.Printf("Error creating Gemini client: %v", err)
		return nil, err
	}

	return &LLMClient{chatModel: model, Key: key}, err
}

// watch out pour le contexte ici
func (o *LLMClient) StartStream(ctx context.Context) context.Context {
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

func (o *LLMClient) StopStream() {
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
func (o *LLMClient) IsRunning() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.running
}

// ClearConversationHistory clears the conversation history, resetting the session state.
// This should be called when starting a new GenerateDocs session.
func (o *LLMClient) ClearConversationHistory() {
	o.conversationHistoryMu.Lock()
	defer o.conversationHistoryMu.Unlock()
	o.conversationHistory = nil
}

// SetListDirectoryBaseRoot binds the list-directory tools to a specific base directory.
// Example: SetListDirectoryBaseRoot("/path/to/project") then tool input "frontend"
// resolves to "/path/to/project/frontend".
func (o *LLMClient) SetListDirectoryBaseRoot(root string) {
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
func (o *LLMClient) loadPrompt(name string) (string, error) {
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

func (o *LLMClient) GenerateDocs(ctx context.Context, req *DocGenerationRequest) (*DocGenerationResponse, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("GenerateDocs: initializing"))

	// Clear any existing conversation history to start fresh
	o.ClearConversationHistory()

	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.DocumentationPath) == "" {
		return nil, fmt.Errorf("documentation path is required")
	}
	if strings.TrimSpace(req.CodebasePath) == "" {
		return nil, fmt.Errorf("codebase path is required")
	}
	docRoot, err := filepath.Abs(req.DocumentationPath)
	if err != nil {
		return nil, err
	}
	codeRoot, err := filepath.Abs(req.CodebasePath)
	if err != nil {
		return nil, err
	}
	o.docRoot = docRoot
	o.codeRoot = codeRoot
	if trimmed := strings.TrimSpace(req.DocumentationRelPath); trimmed != "" {
		o.docRelative = trimmed
	} else {
		o.docRelative = "."
	}
	o.sourceBranch = strings.TrimSpace(req.SourceBranch)
	o.targetBranch = strings.TrimSpace(req.TargetBranch)
	o.sourceCommit = strings.TrimSpace(req.SourceCommit)
	o.targetCommit = ""
	o.SetListDirectoryBaseRoot(docRoot)

	if err := o.prepareSnapshots(ctx); err != nil {
		return nil, err
	}

	docListing, err := o.captureListing(ctx, docRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list documentation root: %w", err)
	}
	codeListing, err := o.captureListing(ctx, codeRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list codebase root: %w", err)
	}

	toolsForSession, err := o.initDocumentationTools(docRoot, codeRoot)
	if err != nil {
		return nil, err
	}

	systemInstr, err := o.loadPrompt("generate_docs.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to load system instructions: %w", err)
	}

	projectInstr, repoErr := o.loadRepoLLMInstructions(docRoot)
	if repoErr != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("unable to load repo LLM instructions: %v", repoErr)))
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: o.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolsForSession,
			},
		},
		Name:          "Documentation Assistant",
		Description:   "Analyzes code diffs and proposes documentation updates",
		Instruction:   systemInstr,
		MaxIterations: 100,
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

	// Section 1: Custom Instructions
	if strings.TrimSpace(projectInstr) != "" {
		promptBuilder.WriteString("# Project-specific documentation instructions\n")
		promptBuilder.WriteString(strings.TrimSpace(projectInstr))
		promptBuilder.WriteString("\n\n")
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("added repo instructions"))
	}

	if strings.TrimSpace(req.SpecificInstr) != "" {
		promptBuilder.WriteString("# Generation-specific documentation instructions\n")
		promptBuilder.WriteString(strings.TrimSpace(req.SpecificInstr))
		promptBuilder.WriteString("\n\n")
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("added specific instructions"))
	}

	// Section 2: Project Context
	promptBuilder.WriteString("# Project Context\n")
	promptBuilder.WriteString(fmt.Sprintf("Project: %s\n", strings.TrimSpace(req.ProjectName)))
	promptBuilder.WriteString(fmt.Sprintf("Source branch: %s\n", strings.TrimSpace(req.SourceBranch)))
	if commit := strings.TrimSpace(req.SourceCommit); commit != "" {
		promptBuilder.WriteString(fmt.Sprintf("Source commit: %s\n", commit))
	}
	promptBuilder.WriteString(fmt.Sprintf("Target branch: %s\n", strings.TrimSpace(req.TargetBranch)))
	promptBuilder.WriteString(fmt.Sprintf("Documentation root: %s\n", filepath.ToSlash(docRoot)))
	promptBuilder.WriteString(fmt.Sprintf("Codebase root: %s\n\n", filepath.ToSlash(codeRoot)))

	// Section 3: Changed Files
	promptBuilder.WriteString("# Changed Files\n")
	promptBuilder.WriteString(changedList)
	promptBuilder.WriteString("\n\n")

	// Section 4: Repository Structure
	promptBuilder.WriteString("# Repository Structure\n\n")
	promptBuilder.WriteString("## Documentation Repository\n")
	promptBuilder.WriteString("<documentation_repo_listing>\n")
	promptBuilder.WriteString(docListing)
	promptBuilder.WriteString("\n</documentation_repo_listing>\n\n")

	promptBuilder.WriteString("## Codebase Repository\n")
	promptBuilder.WriteString("<codebase_repo_listing>\n")
	promptBuilder.WriteString(codeListing)
	promptBuilder.WriteString("\n</codebase_repo_listing>\n\n")

	// Section 5: Code Changes
	promptBuilder.WriteString("# Code Changes\n")
	promptBuilder.WriteString("<git_diff>\n")
	promptBuilder.WriteString(req.Diff)
	promptBuilder.WriteString("\n</git_diff>")

	// Create runner for this generation session
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	println("=== GenerateDocs: Created runner ===")

	iter := runner.Query(ctx, promptBuilder.String())

	var conversationHistory []adk.Message
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
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("GenerateDocs: message error: %v", err)))
			continue
		}
		// Capture the message for conversation history
		conversationHistory = append(conversationHistory, msg)
		lastMessage = msg.Content
	}

	// Store conversation history for potential refinement
	o.conversationHistoryMu.Lock()
	o.conversationHistory = conversationHistory
	o.conversationHistoryMu.Unlock()
	println("=== GenerateDocs: Completed ===")
	println("GenerateDocs: Stored", len(conversationHistory), "messages in conversation history")
	for i, msg := range conversationHistory {
		println("  History", i, "- Role:", msg.Role, "ContentLength:", len(msg.Content), "ToolCalls:", len(msg.ToolCalls))
	}

	events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing complete"))
	return &DocGenerationResponse{Summary: strings.TrimSpace(lastMessage)}, nil
}

func (o *LLMClient) DocRefine(ctx context.Context, req *DocRefineRequest) (*DocGenerationResponse, error) {
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("DocRefine: initializing"))
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.DocumentationPath) == "" {
		return nil, fmt.Errorf("documentation path is required")
	}
	if strings.TrimSpace(req.CodebasePath) == "" {
		return nil, fmt.Errorf("codebase path is required")
	}
	if strings.TrimSpace(req.Instruction) == "" {
		return nil, fmt.Errorf("instruction is required")
	}

	docRoot, err := filepath.Abs(req.DocumentationPath)
	if err != nil {
		return nil, err
	}
	codeRoot, err := filepath.Abs(req.CodebasePath)
	if err != nil {
		return nil, err
	}

	// Set up client state for this session
	o.docRoot = docRoot
	o.codeRoot = codeRoot
	o.sourceBranch = strings.TrimSpace(req.SourceBranch)
	o.targetBranch = ""
	o.sourceCommit = ""
	o.targetCommit = ""
	o.SetListDirectoryBaseRoot(docRoot)

	// Snapshots: live docs; code snapshot not required here
	if err := o.prepareSnapshots(ctx); err != nil {
		return nil, err
	}

	// Check if we have conversation history from GenerateDocs
	o.conversationHistoryMu.Lock()
	conversationHistory := o.conversationHistory
	o.conversationHistoryMu.Unlock()

	println("=== DocRefine: Checking conversation history ===")
	println("conversationHistory length:", len(conversationHistory))

	if len(conversationHistory) > 0 {
		println("DocRefine: will include", len(conversationHistory), "previous messages for context")
	} else {
		println("DocRefine: no conversation history available, starting fresh")
	}

	// Always create a new session for refinement, but include conversation history if available
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("DocRefine: creating refinement session"))

	// Capture repository listings for context
	docListing, err := o.captureListing(ctx, docRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list documentation root: %w", err)
	}
	codeListing, err := o.captureListing(ctx, codeRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list codebase root: %w", err)
	}

	// Initialize documentation-aware tools
	toolsForSession, err := o.initDocumentationTools(docRoot, codeRoot)
	if err != nil {
		return nil, err
	}

	// Load system prompt tailored for refinement
	systemPrompt, err := o.loadPrompt("refine_docs.txt")
	if err != nil {
		return nil, err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: o.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolsForSession,
			},
		},
		Name:        "Documentation Refiner",
		Description: "Applies requested edits to documentation files",
		Instruction: systemPrompt,
	})
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Project: %s\n", strings.TrimSpace(req.ProjectName)))
	if sb := strings.TrimSpace(req.SourceBranch); sb != "" {
		b.WriteString(fmt.Sprintf("Docs branch: %s\n", sb))
	}
	b.WriteString("Documentation repository root:\n")
	b.WriteString(filepath.ToSlash(docRoot))
	b.WriteString("\n\n")
	b.WriteString("Codebase repository root:\n")
	b.WriteString(filepath.ToSlash(codeRoot))
	b.WriteString("\n\n")
	b.WriteString("<documentation_repo_listing>\n")
	b.WriteString(docListing)
	b.WriteString("\n</documentation_repo_listing>\n\n")
	b.WriteString("<codebase_repo_listing>\n")
	b.WriteString(codeListing)
	b.WriteString("\n</codebase_repo_listing>\n\n")
	b.WriteString("<user_instruction>\n")
	b.WriteString(strings.TrimSpace(req.Instruction))
	b.WriteString("\n</user_instruction>")

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	// Build the messages array for the refinement session
	var messages []adk.Message

	if len(conversationHistory) > 0 {
		// Include the previous conversation history for context
		println("DocRefine: Including", len(conversationHistory), "previous messages for context")
		messages = make([]adk.Message, len(conversationHistory))
		copy(messages, conversationHistory)

		// Print the history being included
		for i, msg := range conversationHistory {
			println("  History", i, "- Role:", msg.Role, "ContentLength:", len(msg.Content), "ToolCalls:", len(msg.ToolCalls))
		}
	}

	// Append the new user instruction
	newUserMessage := &schema.Message{
		Role:    schema.User,
		Content: b.String(),
	}
	messages = append(messages, newUserMessage)

	println("DocRefine: Total messages being sent to LLM:", len(messages))
	println("  Last message (new instruction) - Role:", newUserMessage.Role, "ContentLength:", len(newUserMessage.Content))

	// Use Run instead of Query to pass the full message history
	iter := runner.Run(ctx, messages)

	var newMessages []adk.Message
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
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DocRefine: message error: %v", err)))
			continue
		}
		// Capture only the NEW messages from this round (assistant responses)
		newMessages = append(newMessages, msg)
		lastMessage = msg.Content
	}

	// Update conversation history: keep all messages we sent + append new responses
	// messages already contains: [old history + new user message]
	// newMessages contains: [assistant responses from this round]
	o.conversationHistoryMu.Lock()
	o.conversationHistory = append(messages, newMessages...)
	totalMessages := len(o.conversationHistory)
	o.conversationHistoryMu.Unlock()
	println("DocRefine: Updated conversation history, now contains", totalMessages, "messages")
	println("  (sent", len(messages), "messages, got", len(newMessages), "new responses)")

	events.Emit(ctx, events.LLMEventDone, events.NewInfo("LLM processing complete"))
	return &DocGenerationResponse{Summary: strings.TrimSpace(lastMessage)}, nil
}

func (o *LLMClient) prepareSnapshots(ctx context.Context) error {
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
func (o *LLMClient) recordOpenedFile(p string) {
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
func (o *LLMClient) resolveAbsWithinBase(p string) (abs string, err error) {
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
func (o *LLMClient) hasRead(absPath string) bool {
	norm := filepath.ToSlash(strings.TrimSpace(absPath))
	if norm == "" {
		return false
	}
	o.fileHistoryMu.Lock()
	defer o.fileHistoryMu.Unlock()
	return slices.Contains(o.fileOpenHistory, norm)
}

// ResetFileOpenHistory clears the in-memory history for the current client session.
func (o *LLMClient) ResetFileOpenHistory() {
	if o == nil {
		return
	}
	o.fileHistoryMu.Lock()
	o.fileOpenHistory = nil
	o.fileHistoryMu.Unlock()
}

// FileOpenHistory returns a copy of the file-open history for the most recent session.
func (o *LLMClient) FileOpenHistory() []string {
	if o == nil {
		return nil
	}
	o.fileHistoryMu.Lock()
	defer o.fileHistoryMu.Unlock()
	out := make([]string, len(o.fileOpenHistory))
	copy(out, o.fileOpenHistory)
	return out
}

func (o *LLMClient) captureListing(ctx context.Context, root string) (string, error) {
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

func (o *LLMClient) snapshotForRoot(root string) *tools.GitSnapshot {
	if pathsEqual(root, o.codeRoot) {
		return o.codeSnapshot
	}
	return nil
}

func (o *LLMClient) initDocumentationTools(docRoot, codeRoot string) ([]tool.BaseTool, error) {
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

func (o *LLMClient) withBaseRoot(root string, snapshot *tools.GitSnapshot, fn func() error) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("base root not set")
	}
	prevRoot := o.baseRoot
	prevSnapshot := tools.CurrentGitSnapshot()
	prevIgnores := tools.GetScopedIgnorePatterns()
	tools.SetListDirectoryBaseRoot(root)
	tools.SetGitSnapshot(snapshot)
	tools.SetScopedIgnorePatterns(o.scopedIgnoresForRoot(root))
	o.baseRoot = root
	defer func() {
		tools.SetListDirectoryBaseRoot(prevRoot)
		tools.SetGitSnapshot(prevSnapshot)
		tools.SetScopedIgnorePatterns(prevIgnores)
		o.baseRoot = prevRoot
	}()
	return fn()
}

func (o *LLMClient) scopedIgnoresForRoot(root string) []string {
	if strings.TrimSpace(o.docRelative) == "" || o.docRelative == "." {
		return nil
	}
	if o.codeRoot == "" {
		return nil
	}
	if pathsEqual(root, o.codeRoot) {
		rel := filepath.ToSlash(filepath.Clean(o.docRelative))
		rel = strings.TrimPrefix(rel, "./")
		if rel == "" || rel == "." {
			return nil
		}
		return []string{rel, rel + "/**"}
	}
	return nil
}

func (o *LLMClient) resolveToolPath(input string, allowCode bool) (root string, rel string, abs string, err error) {
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

// loadRepoLLMInstructions scans the documentation repository's .narrabyte directory
// for a file beginning with "llm_instructions" and returns its contents.
func (o *LLMClient) loadRepoLLMInstructions(docRoot string) (string, error) {
	docRoot = strings.TrimSpace(docRoot)
	if docRoot == "" {
		return "", nil
	}
	dir := filepath.Join(docRoot, ".narrabyte")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var candidate string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, llmInstructionsNamePrefix) {
			candidate = filepath.Join(dir, name)
			break
		}
	}
	if candidate == "" {
		return "", nil
	}
	data, err := os.ReadFile(candidate)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
