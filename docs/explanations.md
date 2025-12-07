# Narrabyte Architecture & Explanations

This document provides a deep dive into the architecture of Narrabyte, explaining how the Go backend and React frontend work together to deliver AI-powered documentation generation.

## High-Level Overview

Narrabyte is a desktop application built using [Wails](https://wails.io/), which allows writing desktop apps using Go and web technologies.

**Core Mission:** Automate the maintenance of documentation by analyzing code changes between branches and generating corresponding documentation updates.

**The Workflow:**
1.  **Link Repos**: A user links a "Codebase Repo" and a "Documentation Repo" (they can be the same).
2.  **Select Branches**: The user selects a `Source Branch` (e.g., `feature/login`) and a `Target Branch` (e.g., `main`).
3.  **Generate**:
    *   The app calculates the diff between these branches.
    *   It sends the diff + context to an LLM (Claude, GPT-4, Gemini).
    *   The LLM generates updated documentation files.
4.  **Review & Refine**: The user reviews the changes in a dedicated UI, can chat with the AI to refine them, and finally commits them to a new branch in the documentation repo.

---

## Architecture

The application follows a standard Wails architecture:

```mermaid
graph TD
    UI[React Frontend] <-->|Wails Bridge (RPC & Events)| Backend[Go Backend]
    Backend -->|GORM| DB[(SQLite)]
    Backend -->|go-git| Git[Git Operations]
    Backend -->|eino| LLM[LLM Providers]
```

### Key Technologies
*   **Frontend**: React, TypeScript, Vite, TailwindCSS, Zustand.
*   **Backend**: Go, Wails v2.
*   **Database**: SQLite (via `GORM`).
*   **Git**: `go-git` (pure Go implementation of Git).
*   **LLM**: `eino` (Go implementation of OpenAI, Anthropic, Google, etc.)

---

## Backend (Go)

The backend is the brain of the application. It handles all file system operations, git commands, database interactions, and LLM communication.

### Entry Point
*   `main.go`: Initializes the database, creates service instances, and starts the Wails application.
*   `app.go`: Defines the `App` struct and lifecycle hooks (`startup`, `shutdown`).

### Core Services (`internal/services`)

The application logic is divided into services, which are injected into the Wails app and exposed to the frontend.

#### 1. ClientService (`client_service.go`)
This is the most critical service. It orchestrates the entire documentation generation process.
*   **`GenerateDocs`**:
    1.  Prepares the repositories (clones/opens them).
    2.  Calculates the diff between source and target branches.
    3.  Creates a **temporary workspace** for the documentation repo to avoid messing with the user's actual file system during generation.
    4.  Streams the request to the LLM.
    5.  Applies the LLM's suggested changes to the temp workspace.
    6.  Returns the result (changed files, diffs) to the frontend.
*   **`RefineDocs`**: Handles follow-up requests (chat) to modify the generated docs.

#### 2. GitService (`git_service.go`)
A wrapper around `go-git`. It handles:
*   Cloning and opening repositories.
*   Checking out branches.
*   Calculating diffs (`DiffBetweenCommits`, `DiffBetweenBranches`).
*   Managing temporary worktrees.

#### 3. LLM Clients (`internal/llm`)
Abstracts the interaction with different providers (Anthropic, OpenAI, Google).
*   Supports streaming responses.
*   Handles "Thinking" models (e.g., Claude 3.7 Sonnet) and Reasoning models (e.g., o1).

#### 4. Database Services
*   `RepoLinkService`: Manages the links between code and doc repos.
*   `GenerationSessionService`: Stores the history of generation sessions.
*   `AppSettingsService`: Manages global app settings (theme, locale).

### Communication Pattern
*   **RPC**: The frontend calls methods on these services directly (e.g., `GenerateDocs`).
*   **Events**: The backend emits events for real-time updates (e.g., streaming LLM tokens, progress updates).
    *   `event:llm:tool`: Used for streaming text and status updates.
    *   `events:llm:done`: Signals completion.

---

## Frontend (React)

The frontend is a Single Page Application (SPA) responsible for the UI and state management.

### Structure (`frontend/src`)
*   `routes/`: Page components (using TanStack Router concepts, though currently manual routing might be used).
*   `components/`: Reusable UI components.
*   `stores/`: Global state management using **Zustand**.
*   `wailsjs/`: Auto-generated Go bindings.

### State Management: `docGeneration.ts`
This is the most complex store, managing the active generation session.
*   **Session Management**: Keeps track of the current `sessionId`, `status` (idle, running, success), and `result`.
*   **Tab System**: Allows multiple generation sessions to be open in different tabs (`tabSessions`).
*   **Event Listeners**: Sets up listeners for Wails events (`EventsOn`) to update the UI as the LLM streams data.
*   **Actions**:
    *   `start`: Calls `ClientService.GenerateDocs`.
    *   `refine`: Calls `ClientService.RefineDocs`.
    *   `commit`: Calls `ClientService.CommitDocs`.

### Key Components
*   **`ProjectDetailPage`**: The main view for a project. It orchestrates the flow from selecting branches to viewing results.
*   **`DiffViewer`**: Renders the diffs returned by the backend (using `react-diff-view`).
*   **`ChatInterface`**: Allows the user to send refinement instructions to the backend.

---

## Deep Dives

### App Settings Synchronization

This section explains how application settings are synchronized between the frontend and the backend (where "settings" = theme and locale, plus some metadata).

**High-level summary**
- There is a single-row settings model persisted on the backend (ID = 1).
- The frontend keeps a local store (zustand) and calls backend RPCs to read and update settings.
- The frontend initializes itself by calling `Get()` on startup. If the settings have never been persisted, the backend returns default values and the frontend performs a first-run flow that persists a locale choice.
- Updates initiated from the UI call the backend `Update(theme, locale)` RPC; the backend validates and persists the settings, then returns the updated row. The frontend updates its local store and i18n language accordingly.

**Backend Implementation**
- **Model**: `models.AppSettings` (ID, Version, Theme, Locale, UpdatedAt).
- **Repository**: `AppSettingsRepository` ensures ID=1 is always used.
- **Service**: `AppSettingsService` validates inputs and manages the update timestamp.

**Frontend Implementation**
- **Store**: `useAppSettingsStore` in `frontend/src/stores/appSettings.ts`.
- **Initialization**: `init()` checks if `UpdatedAt` is zero. If so, it detects the browser locale and persists it as the default.
- **Normalization**: `normalizeToSupportedLocale` ensures only supported languages (en, fr) are saved.

### Repo Linking

This section explains how repository linking works.

**High-level Summary**
- A `RepoLink` model stores the association: ProjectName, DocumentationRepo, CodebaseRepo.
- The `RepoLinkService` handles creation and retrieval.
- `LinkRepositories` creates the DB record and optionally initializes a Fumadocs project.

**Data Model**
```go
type RepoLink struct {
    ID                uint   `gorm:"primaryKey"`
    DocumentationRepo string
    CodebaseRepo      string
    ProjectName       string
}
```

**Usage Flow**
1.  User provides project name and repo paths.
2.  App calls `repoLinkService.LinkRepositories(...)`.
3.  Service registers the link in SQLite.
4.  Service may trigger `FumadocsService` to set up a documentation site.

### Talking to LLM with Eino

This section explains how Narrabyte communicates with Large Language Models (LLMs) using the [Eino](https://github.com/cloudwego/eino) framework.

**High-level Summary**
- Narrabyte uses `eino` to abstract different LLM providers (OpenAI, Anthropic, Gemini).
- The `LLMClient` struct manages the connection and session state.
- `adk.NewChatModelAgent` creates an agent capable of tool calling.
- `adk.NewRunner` executes the agent and manages the conversation loop.

**Core Components (`internal/llm/client/client.go`)**

1.  **`LLMClient`**: The central struct that holds the `chatModel` (the initialized Eino model) and session-specific data (workspace ID, conversation history).
2.  **`New[Provider]Client`**: Factory functions that initialize the specific Eino chat model (e.g., `claude.NewChatModel`). They handle API keys and model-specific configurations like "thinking" budgets.
3.  **`GenerateDocs` / `DocRefine`**: The main methods that drive the interaction.
    - They prepare the prompt using `buildPromptWithInstructions`.
    - They initialize an `adk.ChatModelAgent` with the model and available tools (like `read_file`, `list_dir`).
    - They create an `adk.Runner` with `EnableStreaming: true`.

**The Streaming Loop**
Narrabyte uses Eino's streaming capabilities to provide real-time feedback. The `runner.Query` (or `runner.Run`) method returns an iterator:

```go
iter := runner.Query(ctx, prompt, ...)
for {
    event, ok := iter.Next()
    if !ok { break }
    // ... handle event ...
    msg, err := o.consumeMessageVariant(ctx, output.MessageOutput)
    // ...
}
```

This loop consumes events from the LLM as they happen. The `consumeMessageVariant` helper (likely internal to `client.go`) processes these events and emits them to the frontend using the `events` package.

### Streaming to Frontend

This section details how the backend streams LLM events (tokens, tool calls, status updates) to the React frontend.

**High-level Summary**
- The backend uses a custom `events` package to emit typed events.
- Wails' `runtime.EventsEmit` bridges these events to the frontend.
- The frontend store (`docGeneration.ts`) subscribes to these events using `EventsOn`.

**Backend: The `events` Package (`internal/events`)**
- **`ToolEvent`**: A struct defining the event payload (`Type`, `Message`, `SessionKey`, `Metadata`).
- **`Emit`**: A global function (configurable via `EnableRuntimeEmitter`) that sends the event.
- **`EnableRuntimeEmitter`**: Configures `Emit` to use Wails' `runtime.EventsEmit`.

**The Flow:**
1.  **LLM Activity**: The `LLMClient` or a tool (e.g., `read_file`) calls `events.Emit(ctx, events.LLMEventTool, events.NewInfo("Reading file..."))`.
2.  **Wails Bridge**: `runtime.EventsEmit` sends this payload to the frontend via the Wails bridge.

**Frontend: Consuming Events (`frontend/src/stores/docGeneration.ts`)**
- **`subscribeToGenerationEvents`**: This function sets up the listeners.
    - `EventsOn("event:llm:tool", ...)`: Listens for general tool events and progress.
    - `EventsOn("events:llm:done", ...)`: Listens for the completion signal.
- **`isEventForSession`**: Ensures that events are routed to the correct session (handling tab-specific sessions).
- **State Update**: When an event is received, it's added to the `events` array in the Zustand store, triggering a UI update (e.g., showing a new log line in the activity feed).