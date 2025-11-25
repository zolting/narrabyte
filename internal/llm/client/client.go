package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"narrabyte/internal/events"
	"narrabyte/internal/llm/tools"
	"narrabyte/internal/utils"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

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

type promptBuilderConfig struct {
	ProjectName   string
	DocRoot       string
	CodeRoot      string
	DocListing    string
	CodeListing   string
	ProjectInstr  string
	SpecificInstr string
	ExtraContext  map[string]string // For additional sections like "Source branch", "Changed Files", etc.
}

// buildPromptWithInstructions constructs a prompt with common sections for documentation tasks
func buildPromptWithInstructions(ctx context.Context, cfg promptBuilderConfig) string {
	var b strings.Builder

	// Section 1: Custom Instructions
	if strings.TrimSpace(cfg.ProjectInstr) != "" {
		b.WriteString("# Project-specific documentation instructions\n")
		b.WriteString(strings.TrimSpace(cfg.ProjectInstr))
		b.WriteString("\n\n")
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("added repo instructions"))
	}

	if strings.TrimSpace(cfg.SpecificInstr) != "" {
		b.WriteString("# Generation-specific documentation instructions\n")
		if strings.Contains(cfg.SpecificInstr, "<DOCUMENTATION_TEMPLATE>") ||
			strings.Contains(cfg.SpecificInstr, "<USER_INSTRUCTIONS>") {
			b.WriteString("The following tagged sections encode user guidance. Apply their content, but do not echo the tags or repeat them in your documentation output.\n\n")
		}
		b.WriteString(strings.TrimSpace(cfg.SpecificInstr))
		b.WriteString("\n\n")
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("added specific instructions"))
	}

	// Section 2: Project Context
	b.WriteString("# Project Context\n")
	b.WriteString(fmt.Sprintf("Project: %s\n", strings.TrimSpace(cfg.ProjectName)))

	// Add extra context fields if provided
	for key, value := range cfg.ExtraContext {
		if strings.TrimSpace(value) != "" {
			b.WriteString(fmt.Sprintf("%s: %s\n", key, strings.TrimSpace(value)))
		}
	}

	b.WriteString(fmt.Sprintf("Documentation repository root: %s\n", filepath.ToSlash(cfg.DocRoot)))
	b.WriteString(fmt.Sprintf("Codebase repository root: %s\n\n", filepath.ToSlash(cfg.CodeRoot)))

	// Section 3: Repository Structure
	b.WriteString("# Repository Structure\n\n")
	b.WriteString("## Documentation Repository\n")
	b.WriteString("<documentation_repo_listing>\n")
	b.WriteString(cfg.DocListing)
	b.WriteString("\n</documentation_repo_listing>\n\n")
	b.WriteString("## Codebase Repository\n")
	b.WriteString("<codebase_repo_listing>\n")
	b.WriteString(cfg.CodeListing)
	b.WriteString("\n</codebase_repo_listing>\n\n")

	return b.String()
}

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
	sessionKey      string
	workspaceID     string

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
	SpecificInstr        string
}

type DocGenerationResponse struct {
	Summary string
}

type docSessionResources struct {
	docListing      string
	codeListing     string
	tools           []tool.BaseTool
	projectInstr    string
	projectInstrErr error
}

type OpenAIModelOptions struct {
	Model           string
	ReasoningEffort string
}

type ClaudeModelOptions struct {
	Model    string
	Thinking bool
}

type GeminiModelOptions struct {
	Model    string
	Thinking bool
}

const (
	reasoningMetadataStreamKey  = "stream"
	reasoningMetadataStreamName = "reasoning"
	reasoningMetadataStateKey   = "state"
	reasoningMetadataReset      = "reset"
	reasoningMetadataUpdate     = "update"
)

func NewOpenAIClient(ctx context.Context, key string, opts OpenAIModelOptions) (*LLMClient, error) {
	modelName := strings.TrimSpace(opts.Model)
	if modelName == "" {
		modelName = "gpt-5-mini"
	}
	var effort openai.ReasoningEffortLevel
	switch strings.ToLower(strings.TrimSpace(opts.ReasoningEffort)) {
	case "low":
		effort = openai.ReasoningEffortLevelLow
	case "high":
		effort = openai.ReasoningEffortLevelHigh
	case "medium":
		effort = openai.ReasoningEffortLevelMedium
	}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:          key,
		Model:           modelName,
		ReasoningEffort: effort,
	})

	if err != nil {
		log.Printf("Error creating OpenAI client: %v", err)
		return nil, err
	}

	return &LLMClient{chatModel: chatModel, Key: key}, err
}

func NewClaudeClient(ctx context.Context, key string, opts ClaudeModelOptions) (*LLMClient, error) {
	modelName := strings.TrimSpace(opts.Model)
	if modelName == "" {
		modelName = "claude-sonnet-4-5"
	}
	chatModel, err := claude.NewChatModel(ctx, &claude.Config{
		APIKey:    key,
		Model:     modelName,
		MaxTokens: 12000,
		Thinking: &claude.Thinking{
			Enable:       opts.Thinking,
			BudgetTokens: 4092,
		},
	})

	if err != nil {
		log.Printf("Error creating Claude client: %v", err)
		return nil, err
	}

	return &LLMClient{chatModel: chatModel, Key: key}, err
}

func NewGeminiClient(ctx context.Context, key string, opts GeminiModelOptions) (*LLMClient, error) {
	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: key,
	})

	if err != nil {
		log.Printf("Error creating Gemini client: %v", err)
		return nil, err
	}

	modelName := strings.TrimSpace(opts.Model)
	if modelName == "" {
		modelName = "gemini-flash-latest"
	}
	var thinkingBudget *int32
	includeThoughts := opts.Thinking
	if !opts.Thinking {
		zero := int32(0)
		thinkingBudget = &zero
	}
	chatModel, err := gemini.NewChatModel(ctx, &gemini.Config{
		Client: genaiClient,
		Model:  modelName,
		ThinkingConfig: &genai.ThinkingConfig{
			IncludeThoughts: includeThoughts,
			ThinkingBudget:  thinkingBudget,
		},
	})

	if err != nil {
		log.Printf("Error creating Gemini client: %v", err)
		return nil, err
	}

	return &LLMClient{chatModel: chatModel, Key: key}, err
}

// watch out pour le contexte ici
func (o *LLMClient) StartStream(ctx context.Context, sessionKey string) context.Context {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return ctx
	}
	o.running = true
	sessionKey = strings.TrimSpace(sessionKey)
	o.sessionKey = sessionKey
	workspaceID := generateSessionID()
	o.workspaceID = workspaceID
	ctx, cancel := context.WithCancel(ctx)
	if sessionKey != "" {
		ctx = events.WithSession(ctx, sessionKey)
	}
	ctx = tools.ContextWithSession(ctx, workspaceID)
	o.cancel = cancel
	o.mu.Unlock()
	return ctx
}

func (o *LLMClient) StopStream() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.workspaceID != "" {
		tools.ClearSession(o.workspaceID)
		tools.ClearTodoSession(o.workspaceID)
		o.workspaceID = ""
	}
	o.sessionKey = ""
	cancel := o.cancel
	if cancel != nil {
		cancel()
	}
	o.running = false
	o.cancel = nil
}

func generateSessionID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
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
	if workspaceID := strings.TrimSpace(o.workspaceID); workspaceID != "" {
		tools.SetGitSnapshotForSession(workspaceID, nil)
		tools.SetListDirectoryBaseRootForSession(workspaceID, abs)
		return
	}
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

func (o *LLMClient) initDocSession(ctx context.Context, docPath string, codePath string, docRel string, sourceBranch string, targetBranch string, sourceCommit string, targetCommit string) (string, string, error) {
	docPath = strings.TrimSpace(docPath)
	codePath = strings.TrimSpace(codePath)
	if docPath == "" {
		return "", "", fmt.Errorf("documentation path is required")
	}
	if codePath == "" {
		return "", "", fmt.Errorf("codebase path is required")
	}

	docRoot, err := filepath.Abs(docPath)
	if err != nil {
		return "", "", err
	}
	codeRoot, err := filepath.Abs(codePath)
	if err != nil {
		return "", "", err
	}

	if trimmed := strings.TrimSpace(docRel); trimmed != "" {
		o.docRelative = trimmed
	} else {
		o.docRelative = "."
	}
	o.docRoot = docRoot
	o.codeRoot = codeRoot
	o.sourceBranch = strings.TrimSpace(sourceBranch)
	o.targetBranch = strings.TrimSpace(targetBranch)
	o.sourceCommit = strings.TrimSpace(sourceCommit)
	o.targetCommit = strings.TrimSpace(targetCommit)
	o.SetListDirectoryBaseRoot(docRoot)

	if err := o.prepareSnapshots(ctx); err != nil {
		return "", "", err
	}
	return docRoot, codeRoot, nil
}

func (o *LLMClient) prepareDocResources(ctx context.Context, docRoot string, codeRoot string) (*docSessionResources, error) {
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

	projectInstr, repoErr := o.loadRepoLLMInstructions(docRoot)

	return &docSessionResources{
		docListing:      docListing,
		codeListing:     codeListing,
		tools:           toolsForSession,
		projectInstr:    projectInstr,
		projectInstrErr: repoErr,
	}, nil
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
	docRoot, codeRoot, err := o.initDocSession(ctx, req.DocumentationPath, req.CodebasePath, req.DocumentationRelPath, req.SourceBranch, req.TargetBranch, req.SourceCommit, "")
	if err != nil {
		return nil, err
	}

	resources, err := o.prepareDocResources(ctx, docRoot, codeRoot)
	if err != nil {
		return nil, err
	}

	systemInstr, err := o.loadPrompt("generate_docs.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to load system instructions: %w", err)
	}

	if resources.projectInstrErr != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("unable to load repo LLM instructions: %v", resources.projectInstrErr)))
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: o.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: resources.tools,
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

	extraContext := map[string]string{
		"Source branch": req.SourceBranch,
		"Target branch": req.TargetBranch,
	}
	if commit := strings.TrimSpace(req.SourceCommit); commit != "" {
		extraContext["Source commit"] = commit
	}

	prompt := buildPromptWithInstructions(ctx, promptBuilderConfig{
		ProjectName:   req.ProjectName,
		DocRoot:       docRoot,
		CodeRoot:      codeRoot,
		DocListing:    resources.docListing,
		CodeListing:   resources.codeListing,
		ProjectInstr:  resources.projectInstr,
		SpecificInstr: req.SpecificInstr,
		ExtraContext:  extraContext,
	})

	var promptBuilder strings.Builder
	promptBuilder.WriteString(prompt)

	// Section 4: Changed Files
	promptBuilder.WriteString("# Changed Files\n")
	promptBuilder.WriteString(changedList)
	promptBuilder.WriteString("\n\n")

	// Section 5: Code Changes
	promptBuilder.WriteString("# Code Changes\n")
	promptBuilder.WriteString("<git_diff>\n")
	promptBuilder.WriteString(req.Diff)
	promptBuilder.WriteString("\n</git_diff>")

	// Create runner for this generation session
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true})
	println("=== GenerateDocs: Created runner ===")

	// Store the user query as the first message in conversation history
	// This ensures when history is restored, the first message is always a user message
	userQueryMessage := &schema.Message{
		Role:    schema.User,
		Content: promptBuilder.String(),
	}

	iter := runner.Query(ctx, promptBuilder.String())

	// Initialize conversation history with the user query
	conversationHistory := []adk.Message{userQueryMessage}
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

		output := event.Output
		if output == nil || output.MessageOutput == nil {
			continue
		}
		msg, err := o.consumeMessageVariant(ctx, output.MessageOutput)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("GenerateDocs: message error: %v", err)))
			continue
		}
		if msg == nil {
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

	docRoot, codeRoot, err := o.initDocSession(ctx, req.DocumentationPath, req.CodebasePath, req.DocumentationRelPath, req.SourceBranch, "", "", "")
	if err != nil {
		return nil, err
	}

	// Always create a new session for refinement, but include conversation history if available
	events.Emit(ctx, events.LLMEventTool, events.NewInfo("DocRefine: creating refinement session"))

	resources, err := o.prepareDocResources(ctx, docRoot, codeRoot)
	if err != nil {
		return nil, err
	}

	// Load system prompt tailored for refinement
	systemPrompt, err := o.loadPrompt("refine_docs.txt")
	if err != nil {
		return nil, err
	}

	if resources.projectInstrErr != nil {
		events.Emit(ctx, events.LLMEventTool, events.NewWarn(fmt.Sprintf("unable to load repo LLM instructions: %v", resources.projectInstrErr)))
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: o.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: resources.tools,
			},
		},
		Name:        "Documentation Refiner",
		Description: "Applies requested edits to documentation files",
		Instruction: systemPrompt,
	})
	if err != nil {
		return nil, err
	}

	extraContext := map[string]string{}
	if sb := strings.TrimSpace(req.SourceBranch); sb != "" {
		extraContext["Docs branch"] = sb
	}

	prompt := buildPromptWithInstructions(ctx, promptBuilderConfig{
		ProjectName:   req.ProjectName,
		DocRoot:       docRoot,
		CodeRoot:      codeRoot,
		DocListing:    resources.docListing,
		CodeListing:   resources.codeListing,
		ProjectInstr:  resources.projectInstr,
		SpecificInstr: req.SpecificInstr,
		ExtraContext:  extraContext,
	})

	conversationHistory, historyAdjusted := o.conversationHistoryForRun(prompt)

	println("=== DocRefine: Checking conversation history ===")
	println("conversationHistory length:", len(conversationHistory))

	if len(conversationHistory) > 0 {
		println("DocRefine: will include", len(conversationHistory), "previous messages for context")
	} else {
		println("DocRefine: no conversation history available, starting fresh")
	}

	if historyAdjusted {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("DocRefine: normalized stored conversation history"))
	}

	var b strings.Builder
	b.WriteString(prompt)

	// Section 4: User Instruction
	b.WriteString("# User Refinement Request\n\n")
	b.WriteString("The user is requesting specific changes to the documentation. ")
	b.WriteString("Focus on applying the requested edits directly. ")
	b.WriteString("You have access to both repositories if needed, but prioritize making the requested documentation changes efficiently.\n\n")
	b.WriteString("<user_instruction>\n")
	b.WriteString(strings.TrimSpace(req.Instruction))
	b.WriteString("\n</user_instruction>")

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true})

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
		output := event.Output
		if output == nil || output.MessageOutput == nil {
			continue
		}
		msg, err := o.consumeMessageVariant(ctx, output.MessageOutput)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DocRefine: message error: %v", err)))
			continue
		}
		if msg == nil {
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

func (o *LLMClient) consumeMessageVariant(ctx context.Context, mv *adk.MessageVariant) (*schema.Message, error) {
	if mv == nil {
		return nil, nil
	}
	if mv.IsStreaming {
		if mv.MessageStream == nil {
			o.broadcastReasoningContent(ctx, mv.Message)
			return mv.Message, nil
		}
		return o.consumeStreamingMessage(ctx, mv.MessageStream)
	}
	msg, err := mv.GetMessage()
	if err != nil {
		return nil, err
	}
	o.broadcastReasoningContent(ctx, msg)
	return msg, nil
}

func (o *LLMClient) consumeStreamingMessage(ctx context.Context, stream adk.MessageStream) (*schema.Message, error) {
	if stream == nil {
		return nil, nil
	}
	stream.SetAutomaticClose()
	defer stream.Close()

	var (
		chunks           []*schema.Message
		reasoningBuilder strings.Builder
		hasReasoning     bool
	)

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("stream receive error: %w", err)
		}
		if chunk == nil {
			continue
		}
		if chunk.ReasoningContent != "" {
			if !hasReasoning {
				o.emitReasoningReset(ctx)
				hasReasoning = true
			}
			_, _ = reasoningBuilder.WriteString(chunk.ReasoningContent)
			o.emitReasoningUpdate(ctx, reasoningBuilder.String())
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return nil, nil
	}

	msg, err := schema.ConcatMessages(chunks)
	if err != nil {
		return nil, err
	}
	if !hasReasoning {
		o.broadcastReasoningContent(ctx, msg)
	}
	return msg, nil
}

func (o *LLMClient) broadcastReasoningContent(ctx context.Context, msg *schema.Message) {
	if msg == nil {
		return
	}
	if strings.TrimSpace(msg.ReasoningContent) == "" {
		return
	}
	o.emitReasoningReset(ctx)
	o.emitReasoningUpdate(ctx, msg.ReasoningContent)
}

func (o *LLMClient) emitReasoningReset(ctx context.Context) {
	evt := events.NewSuccess("")
	evt.Metadata = map[string]string{
		reasoningMetadataStreamKey: reasoningMetadataStreamName,
		reasoningMetadataStateKey:  reasoningMetadataReset,
	}
	events.Emit(ctx, events.LLMEventTool, evt)
}

func (o *LLMClient) emitReasoningUpdate(ctx context.Context, content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	evt := events.NewSuccess(content)
	evt.Metadata = map[string]string{
		reasoningMetadataStreamKey: reasoningMetadataStreamName,
		reasoningMetadataStateKey:  reasoningMetadataUpdate,
	}
	events.Emit(ctx, events.LLMEventTool, evt)
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
		err = o.withBaseRoot(root, snapshot, func() error {
			res, innerErr := tools.ListDirectory(ctx, &tools.ListLSInput{Path: rel, Ignore: ignore})
			if innerErr != nil {
				events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ListDirectory(policy): %v", innerErr)))
				return innerErr
			}
			output = res.Output
			return nil
		})

		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("tool=list_directory_tool path=%s", filepath.ToSlash(rel))))
		} else {
			events.Emit(ctx, events.LLMEventTool, events.NewSuccess(fmt.Sprintf("tool=list_directory_tool path=%s", filepath.ToSlash(rel))))
		}

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
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("ReadFile(policy): %v path= %s", err, filepath.ToSlash(rel))))
			return out, err
		}
		if out != nil && (out.Metadata == nil || out.Metadata["error"] == "") {
			o.recordOpenedFile(abs)
		}

		events.Emit(ctx, events.LLMEventTool, events.NewSuccess(fmt.Sprintf("tool=read_file_tool path=%s", filepath.ToSlash(rel))))
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
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("WriteFile(policy): resolving '%s'", p)))
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

		events.Emit(ctx, events.LLMEventTool, events.NewSuccess(fmt.Sprintf("tool=write_file_tool path=%s", title)))
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
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("EditFile(policy): resolving '%s'", p)))
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

		events.Emit(ctx, events.LLMEventTool, events.NewSuccess(fmt.Sprintf("tool=edit_file_tool path=%s", title)))
		return out, nil
	}
	editTool, err := einoUtils.InferTool("edit_tool", editDesc, editWithPolicy)
	if err != nil {
		return nil, err
	}

	// TodoWrite tool - manage task list for documentation generation
	todoWriteDesc := tools.ToolDescription("todo_write_tool")
	if strings.TrimSpace(todoWriteDesc) == "" {
		todoWriteDesc = "REPLACE the entire task list for the current session (read current list first, then send complete updated list)"
	}
	todoWriteWithPolicy := func(ctx context.Context, in *tools.TodoWriteInput) (*tools.TodoWriteOutput, error) {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("TodoWrite: updating task list"))

		// Check if we're reducing the todo count (potential accidental deletion)
		sessionID := tools.SessionIDFromContext(ctx)
		if sessionID != "" {
			session := tools.GetTodoSession(sessionID)
			existing := session.GetTodos()
			if len(existing) > 0 && len(in.Todos) < len(existing) {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn(
					fmt.Sprintf("TodoWrite: Reducing todo count from %d to %d tasks. Ensure this is intentional - remember this tool replaces the entire list.",
						len(existing), len(in.Todos))))
			}
		}

		out, err := tools.WriteTodo(ctx, in)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("TodoWrite: error: %v", err)))
			return nil, err
		}
		// Emit the todo update as a special event with metadata
		if out.Metadata != nil {
			if todos, ok := out.Metadata["todos"].([]tools.Todo); ok {
				events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("TodoWrite: %s", out.Title)))
				// Convert tools.Todo to events.TodoItem for emission
				todoItems := make([]events.TodoItem, len(todos))
				for i, todo := range todos {
					todoItems[i] = events.TodoItem{
						Content:    todo.Content,
						ActiveForm: todo.ActiveForm,
						Status:     string(todo.Status),
					}
				}
				// Emit a special todo update event that frontend can listen to
				events.EmitTodoUpdate(ctx, todoItems)
			}
		}

		events.Emit(ctx, events.LLMEventTool, events.NewSuccess(fmt.Sprintf("TodoWrite: %s", out.Title)))
		return out, nil
	}
	todoWriteTool, err := einoUtils.InferTool("todo_write_tool", todoWriteDesc, todoWriteWithPolicy)
	if err != nil {
		return nil, err
	}

	// TodoRead tool - read the current task list
	todoReadDesc := tools.ToolDescription("todo_read_tool")
	if strings.TrimSpace(todoReadDesc) == "" {
		todoReadDesc = "read the current task list (ALWAYS call this before todo_write_tool to avoid deleting tasks)"
	}
	todoReadWithPolicy := func(ctx context.Context, in *tools.TodoReadInput) (*tools.TodoReadOutput, error) {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("TodoRead: reading task list"))
		// Handle nil input (tool called with no arguments)
		if in == nil {
			in = &tools.TodoReadInput{}
		}
		out, err := tools.ReadTodo(ctx, in)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("TodoRead: error: %v", err)))
			return nil, err
		}

		events.Emit(ctx, events.LLMEventTool, events.NewSuccess(fmt.Sprintf("TodoRead: %s", out.Title)))
		return out, nil
	}
	todoReadTool, err := einoUtils.InferTool("todo_read_tool", todoReadDesc, todoReadWithPolicy)
	if err != nil {
		return nil, err
	}

	deleteDesc := tools.ToolDescription("delete_file_tool")
	if strings.TrimSpace(deleteDesc) == "" {
		deleteDesc = "delete a file from the documentation repository"
	}
	deleteWithPolicy := func(ctx context.Context, in *tools.DeleteFileInput) (*tools.DeleteFileOutput, error) {
		events.Emit(ctx, events.LLMEventTool, events.NewInfo("DeleteFile(policy): starting"))
		if in == nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError("DeleteFile(policy): input is required"))
			return &tools.DeleteFileOutput{
				Output:   "Format error: input is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		p := strings.TrimSpace(in.FilePath)
		if p == "" {
			events.Emit(ctx, events.LLMEventTool, events.NewError("DeleteFile(policy): file_path is required"))
			return &tools.DeleteFileOutput{
				Output:   "Format error: file_path is required",
				Metadata: map[string]string{"error": "format_error"},
			}, nil
		}
		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("DeleteFile(policy): resolving '%s'", p)))
		absCandidate, rerr := o.resolveAbsWithinBase(p)
		if rerr != nil {
			if rerr.Error() == "project root not set" {
				events.Emit(ctx, events.LLMEventTool, events.NewError("DeleteFile(policy): documentation root not set"))
				return &tools.DeleteFileOutput{
					Output:   "Format error: project root not set",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			if strings.Contains(rerr.Error(), "escapes") {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("DeleteFile(policy): path escapes the documentation root"))
				return &tools.DeleteFileOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Format error: path escapes the documentation root",
					Metadata: map[string]string{"error": "format_error"},
				}, nil
			}
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile(policy): resolve error: %v", rerr)))
			return nil, rerr
		}

		if st, err := os.Stat(absCandidate); err == nil && !st.IsDir() {
			if !o.hasRead(absCandidate) {
				events.Emit(ctx, events.LLMEventTool, events.NewWarn("DeleteFile(policy): policy violation - must read before delete"))
				return &tools.DeleteFileOutput{
					Title:    filepath.ToSlash(absCandidate),
					Output:   "Policy error: must read the file before deleting",
					Metadata: map[string]string{"error": "policy_violation"},
				}, nil
			}
		}

		events.Emit(ctx, events.LLMEventTool, events.NewInfo(fmt.Sprintf("DeleteFile(policy): invoking underlying DeleteFile for '%s'", filepath.ToSlash(absCandidate))))
		out, err := tools.DeleteFile(ctx, in)
		if err != nil {
			events.Emit(ctx, events.LLMEventTool, events.NewError(fmt.Sprintf("DeleteFile(policy): underlying error: %v", err)))
			return out, err
		}
		title := ""
		if out != nil && strings.TrimSpace(out.Title) != "" {
			title = filepath.ToSlash(out.Title)
		}

		events.Emit(ctx, events.LLMEventTool, events.NewSuccess(fmt.Sprintf("tool=delete_file_tool path=%s", title)))
		return out, nil
	}
	deleteTool, err := einoUtils.InferTool("delete_file_tool", deleteDesc, deleteWithPolicy)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{listTool, readTool, writeTool, editTool, todoWriteTool, todoReadTool, deleteTool}, nil
}

func (o *LLMClient) withBaseRoot(root string, snapshot *tools.GitSnapshot, fn func() error) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("base root not set")
	}
	workspaceID := strings.TrimSpace(o.workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("session not initialized")
	}
	prevSessionRoot := tools.ListDirectoryBaseRootForSession(workspaceID)
	prevSnapshot := tools.GitSnapshotForSession(workspaceID)
	prevIgnores := tools.GetScopedIgnorePatternsForSession(workspaceID)
	prevBase := o.baseRoot
	tools.SetListDirectoryBaseRootForSession(workspaceID, root)
	tools.SetGitSnapshotForSession(workspaceID, snapshot)
	tools.SetScopedIgnorePatternsForSession(workspaceID, o.scopedIgnoresForRoot(root))
	o.baseRoot = root
	defer func() {
		tools.SetListDirectoryBaseRootForSession(workspaceID, prevSessionRoot)
		tools.SetGitSnapshotForSession(workspaceID, prevSnapshot)
		tools.SetScopedIgnorePatternsForSession(workspaceID, prevIgnores)
		o.baseRoot = prevBase
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

type persistableMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ConversationHistoryJSON returns a compact JSON array of messages containing
// only role and content, suitable for persistence.
func (o *LLMClient) ConversationHistoryJSON() (string, error) {
	o.conversationHistoryMu.Lock()
	defer o.conversationHistoryMu.Unlock()
	msgs := make([]persistableMessage, 0, len(o.conversationHistory))
	for _, m := range o.conversationHistory {
		// Skip messages with empty content (e.g., tool-call-only messages)
		// since we can't properly restore them without tool call details
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		msgs = append(msgs, persistableMessage{Role: string(m.Role), Content: m.Content})
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// LoadConversationHistoryJSON restores conversation history from a JSON array
// created by ConversationHistoryJSON. It replaces any existing history.
func (o *LLMClient) LoadConversationHistoryJSON(jsonStr string) error {
	if strings.TrimSpace(jsonStr) == "" {
		return nil
	}
	var msgs []persistableMessage
	if err := json.Unmarshal([]byte(jsonStr), &msgs); err != nil {
		return err
	}
	o.conversationHistoryMu.Lock()
	defer o.conversationHistoryMu.Unlock()
	o.conversationHistory = nil

	var history []adk.Message
	for _, pm := range msgs {
		// Skip messages with empty content
		if strings.TrimSpace(pm.Content) == "" {
			continue
		}

		msg := &schema.Message{Role: schema.RoleType(pm.Role), Content: pm.Content}
		history = append(history, msg)
	}

	// Validate that first message is a user message (required by Anthropic and good practice for all providers)
	if len(history) > 0 && history[0].Role != schema.User {
		return fmt.Errorf("invalid conversation history: first message must be a user message (got %s)", history[0].Role)
	}

	o.conversationHistory = history
	return nil
}

// HasConversationHistory reports whether any conversation history is present.
func (o *LLMClient) HasConversationHistory() bool {
	o.conversationHistoryMu.Lock()
	defer o.conversationHistoryMu.Unlock()
	return len(o.conversationHistory) > 0
}

func (o *LLMClient) conversationHistoryForRun(fallbackFirstUser string) ([]adk.Message, bool) {
	o.conversationHistoryMu.Lock()
	defer o.conversationHistoryMu.Unlock()

	if len(o.conversationHistory) == 0 {
		return nil, false
	}

	normalized, changed := normalizeConversationHistory(o.conversationHistory, fallbackFirstUser)
	if changed {
		o.conversationHistory = normalized
	}

	snapshot := make([]adk.Message, len(o.conversationHistory))
	copy(snapshot, o.conversationHistory)
	return snapshot, changed
}

func normalizeConversationHistory(history []adk.Message, fallbackFirstUser string) ([]adk.Message, bool) {
	if len(history) == 0 {
		return history, false
	}

	firstUserIdx := -1
	for i, msg := range history {
		if msg.Role == schema.User {
			firstUserIdx = i
			break
		}
	}

	if firstUserIdx == 0 {
		return history, false
	}

	normalized := make([]adk.Message, 0, len(history)+1)
	changed := false

	if firstUserIdx == -1 {
		trimmed := strings.TrimSpace(fallbackFirstUser)
		if trimmed == "" {
			trimmed = "Previous session restored without the original user prompt. Continue the documentation workflow from the latest assistant response."
		}
		normalized = append(normalized, &schema.Message{Role: schema.User, Content: trimmed})
		normalized = append(normalized, history...)
		return normalized, true
	}

	for _, msg := range history[:firstUserIdx] {
		if msg.Role == schema.System {
			normalized = append(normalized, msg)
			continue
		}
		changed = true
	}

	normalized = append(normalized, history[firstUserIdx:]...)
	if !changed && len(normalized) == len(history) {
		return history, false
	}
	return normalized, true
}
