# App Settings Synchronization

This document explains how application settings are synchronized between the frontend and the backend in this project (where "settings" = theme and locale, plus some metadata).

High-level summary

- There is a single-row settings model persisted on the backend (ID = 1).
- The frontend keeps a local store (zustand) and calls backend RPCs (Wails-generated bindings) to read and update settings.
- The frontend initializes itself by calling Get() on startup. If the settings have never been persisted, the backend returns default values and the frontend performs a first-run flow that persists a locale choice.
- Updates initiated from the UI call the backend Update(theme, locale) RPC; the backend validates and persists the settings, then returns the updated row. The frontend updates its local store and i18n language accordingly.

Backend (Go)

Relevant files:
- internal/models/app_settings.go
- internal/repositories/app_settings_repository.go
- internal/services/app_settings_service.go

Model
- models.AppSettings is a simple struct with fields: ID (primary key), Version, Theme, Locale, UpdatedAt (ISO string).
- The design expects a single-row table where the settings row always has ID = 1.

Repository
- AppSettingsRepository.Get(ctx) reads the row with primary key 1.
- If the record is not found, Get returns a default AppSettings object:
  - ID: 1
  - Version: 1
  - Theme: "system"
  - Locale: "en"
  - UpdatedAt: "" (empty string used to represent zero/unpersisted time)
- AppSettingsRepository.Update(ctx, settings) forces settings.ID = 1 and saves the row using GORM Save().

Service
- AppSettingsService exposes Get() and Update(theme, locale) to callers.
- Update validates inputs:
  - theme must be "light", "dark" or "system" (errors otherwise)
  - locale must be non-empty
- Update fetches the current settings (via repository Get), mutates Theme and Locale and sets UpdatedAt = time.Now().Format(time.RFC3339), then calls repository.Update to persist the row and returns the updated struct.
- Tests show error cases covered (missing theme/locale, invalid theme, repo errors on Get/Update).

Frontend (TypeScript / React)

Relevant files:
- frontend/src/stores/appSettings.ts
- frontend/src/routes/settings.tsx (UI that consumes the store)
- frontend uses Wails-generated bindings: Get and Update from wailsjs/go/services/appSettingsService

Local store
- useAppSettingsStore (zustand) holds the local state: settings | null, initialized, loading, error.
- Methods available in the store:
  - init(): called to bootstrap the store (calls backend Get())
  - setTheme(theme): calls Update(theme, locale) on the backend and updates the store with the returned settings
  - setLocale(locale): calls Update(theme, normalizedLocale) and updates the store + i18n language
  - update(theme, locale): helper to do both and set i18n language

Initialization and "first run" logic
- init() calls Get() (backend RPC).
- The frontend function isZeroTimeISOString checks if settings.UpdatedAt is effectively a zero/unset value. It treats empty, non-parseable dates, non-positive timestamps and Go's zero date string ("0001-01-01...") as zero-time.
- If UpdatedAt is zero-time, the frontend assumes this is the first run (or settings have not been persisted yet). It tries to detect the user's language preference (i18n.language or navigator.language), normalizes it, and then calls Update(settings.Theme ?? "system", detectedLocale) to persist the detected locale.
  - After Update returns, the frontend sets the store { settings: updated, initialized: true, loading: false } and ensures i18n.language is the detected locale.
- If UpdatedAt is not zero-time (settings already persisted), the frontend simply applies the persisted locale to i18n if needed and sets the store.

Locale normalization
- normalizeToSupportedLocale reduces an input like "en-US" to its base ("en") and only supports "en" and "fr". Anything not starting with "fr" becomes "en". This mirrors the app's supported languages.

UI
- The settings page (frontend/src/routes/settings.tsx) uses useAppSettingsStore to read settings, show current theme & language, and call setTheme/setLocale when the user clicks buttons.
- The store methods call Update(theme, locale) RPC and then update the local store with the response. The Update RPC returns the persisted settings.

Synchronization guarantees and notes
- Backend is the source of truth: Update persists changes and returns the canonical row which the frontend stores locally.
- Single-row model: repository.Update always sets ID = 1 and uses GORM Save(). There is no explicit optimistic concurrency control shown (Version exists in the model but is not used in Update). Concurrent updates may overwrite each other; if that is a concern, Version-based optimistic locking or database transactions should be added.
- UpdatedAt is set on every Update to the current server time (time.Now().Format(time.RFC3339)) which the frontend uses to detect whether the settings were persisted before.
- The frontend performs normalization and language switching locally (i18n.changeLanguage) after Update returns; the backend does not try to transform locales beyond persisting what is sent.

Errors and edge cases
- Backend Update validates theme values; an invalid input will return an error which the frontend code should surface (store.error). Tests cover these cases.
- If repository.Get returns gorm.ErrRecordNotFound, the repository returns the default values (so the frontend will receive a default settings struct on first run).
- The frontend treats any non-parseable UpdatedAt as zero-time and enters the first-run flow.

Implementation flow example (first run)
1. App starts and frontend calls useAppSettingsStore.init().
2. Frontend calls backend Get() via Wails binding.
3. Repo has no row -> Get() returns default settings with UpdatedAt = "".
4. Frontend detects zero UpdatedAt and determines the detected locale from i18n or navigator.
5. Frontend calls backend Update(theme, detectedLocale).
6. Backend Update validates, reads current row (default), sets Theme and Locale and UpdatedAt to now, persists via repository.Update (ID=1), returns the updated struct.
7. Frontend receives the updated struct, sets the local store and changes i18n.language if needed.

Implementation flow example (user changes theme)
1. User clicks "Dark" in the settings UI; settings.tsx calls setTheme("dark") from the store.
2. Store setTheme determines the current locale to send, calls Update("dark", locale).
3. Backend Update validates, updates the DB row with the new Theme and UpdatedAt, returns updated settings.
4. Frontend updates the store with the returned settings; UI reflects the change.

Where to look in code
- Backend logic: internal/services/app_settings_service.go and internal/repositories/app_settings_repository.go
- Model: internal/models/app_settings.go
- Frontend store and sync logic: frontend/src/stores/appSettings.ts
- Frontend UI: frontend/src/routes/settings.tsx

Suggestions / considerations
- If concurrent updates are a concern, implement optimistic locking using Version or database transactions.
- Consider returning the UpdatedAt from the backend and letting the frontend use server time as the source of truth (it already does), but be explicit about timezone format expectations.
- If more locales will be added, keep normalizeToSupportedLocale in sync with i18n supported languages.## Git Diff Frontend Component

What it is (short)

- Summary: Renders unified git diffs in a dialog using react-diff-view; parses with parseDiff and supports split/unified views. The component is designed to accept a diff string (currently it uses SAMPLE_DIFF) and render per-file hunks with a toggleable view type.

- A small, reusable Git diff viewer exposed as a Dialog: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx. It parses unified diff text and renders it with the react-diff-view library inside the app's Dialog primitive. The component currently ships with a hardcoded SAMPLE_DIFF used as a placeholder; it is structured so a real diff string can be passed in with minimal changes (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L1-L96).

Quick summary (practical)

- Input: a unified-diff string (SAMPLE_DIFF by default).
- Parse: parseDiff turns the string into an array of file objects (oldPath/newPath, hunks).
- Render: react-diff-view's <Diff> + <Hunk> render line-level additions/removals; local CSS styles it.
- Controls: toolbar allows toggling between "split" and "unified" views; the trigger is supplied by the caller via DialogTrigger asChild.

How it works (implementation)

- Files and imports:
  - frontend/src/components/GitDiffDialog/GitDiffDialog.tsx — main component, imports parseDiff, Diff and Hunk from react-diff-view, app Dialog/Button primitives and CSS (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L1-L13).
  - frontend/src/components/GitDiffDialog/diff-view-theme.css — local styling layered on top of the library stylesheet (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L4-L5).
  - The component is used on the home route where a compact icon Button is wrapped by <GitDiffDialog> to act as the trigger (source: frontend/src/routes/index.tsx#L114-L119).

- Data flow and parsing:
  - The component defines SAMPLE_DIFF (placeholder unified-diff string) and calls parseDiff(SAMPLE_DIFF) to convert it into an array of file objects (each with metadata and hunks). Parsing is memoized with useMemo to avoid repeated work (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L15-L33, frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L43-L46).
  - The current implementation selects the first parsed file (files[0]) and renders its newPath and hunks. If parsing yields no files the UI falls back to showing "example.js" as the filename (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L44-L46, frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L62-L65).

- Rendering and interactivity:
  - The component is implemented as a Dialog. DialogTrigger is used with asChild so any child passed into <GitDiffDialog> (for example the Button on the home route) becomes the clickable trigger opening the dialog (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L54-L56).
  - A local React state viewType toggles between "split" and "unified" rendering modes. The toolbar Button toggles this state; labels come from i18n keys (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L41-L51, frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L66-L75).
  - The react-diff-view <Diff> component receives diffType, hunks and viewType, and renders each hunk by mapping to <Hunk>. Lines inherit styling from the library CSS plus the local diff-view-theme.css (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L77-L89, frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L4-L5).

Implementation details (step-by-step)

1. On initial render the component memoizes parseDiff(SAMPLE_DIFF) so parsing occurs once (or when the input changes). The parsed result is an array of file objects with fields like oldPath/newPath, type and hunks (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L43-L46).

2. The UI displays the filename from file.newPath (or a fallback) and shows a toolbar Button to toggle viewType between "split" and "unified". The button label is internationalized (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L62-L75).

3. The <Diff> component from react-diff-view receives the parsed hunks and current viewType; the component maps hunks to <Hunk> elements which render line-level additions/deletions/context (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L77-L89).

4. Because DialogTrigger uses asChild, the consumer controls the trigger element and its accessible label. The home route provides an sr-only label on the trigger Button for screen readers (source: frontend/src/routes/index.tsx#L115-L118).

How to extend it to show real diffs (recommended approach)

- Prop-based input (recommended): Make GitDiffDialog accept a prop such as diff?: string and memoize parseDiff(diff ?? SAMPLE_DIFF). This keeps parsing client-side and keeps the existing trigger-as-child API.

  Example (pseudo):
  - function GitDiffDialog({ children, diff }: { children: React.ReactNode; diff?: string }) {
  -   const parsed = useMemo(() => parseDiff(diff ?? SAMPLE_DIFF), [diff]);
  -   ...
  - }

- Fetching diffs from the backend (implementation note): Add or extend a Git RPC in the existing Wails GitService that runs git --no-pager diff -- <path> (or similar) and returns the unified diff string. The frontend can call that RPC, pass the returned string into GitDiffDialog as the diff prop, and render it client-side.
  - The repo already imports a Wails GitService in the routes file and Wails bindings exist under frontend/wailsjs/go/services/GitService.* (source: frontend/src/routes/index.tsx#L12-L12, frontend/wailsjs/go/services/GitService.d.ts#L1-L1) — this usage is inferred from imports in the code (Inferred).

Practical notes and UX improvements (Inferred)

- Add a loading state while fetching diffs and display a friendly placeholder when the diff is empty.
- Surface parsing errors if parseDiff throws or returns an unexpected shape.
- When parseDiff returns multiple changed files, show a file list / tabs instead of only the first file (current behavior) to improve discoverability.
- Preserve scroll/split-pane sync and add keyboard shortcuts for accessibility.

i18n & accessibility

- The dialog title and toggle labels use i18n keys (e.g., t("common.gitDiff") and t("common.splitView")), so translations will apply when keys exist in locale files (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L40-L41, frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L58-L60, frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L72-L74).
- The component delegates accessible labeling to the trigger element (the home route supplies an sr-only label), which keeps the Dialog semantics straightforward (source: frontend/src/routes/index.tsx#L115-L118).

Usage (quick example)

- Minimal usage (current pattern):
  - <GitDiffDialog>
  -   <Button aria-label="Open diff">...</Button>
  - </GitDiffDialog>
  - This uses SAMPLE_DIFF internally and opens the dialog when the Button is clicked.

- Preferred usage (pass a real diff string):
  - const diff = await GitService.GetDiff(path)
  - <GitDiffDialog diff={diff}>
  -   <Button aria-label="Open diff">...</Button>
  - </GitDiffDialog>

Sources

- Component implementation: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L1-L96
- SAMPLE_DIFF definition and parse usage: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L15-L46
- Diff rendering and view toggle: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L41-L51, frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L66-L89
- Dialog trigger usage (asChild): frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L54-L56
- Usage in home route (trigger): frontend/src/routes/index.tsx#L114-L119
- Wails GitService import (example of existing git bindings): frontend/src/routes/index.tsx#L12-L12, frontend/wailsjs/go/services/GitService.d.ts#L1-L1 (Inferred)
# Repo Linking

This section explains how repository linking works in this project. Repo linking associates a documentation repository (e.g., for Markdown docs) with a codebase repository for a given project, enabling features like automatic documentation setup using Fumadocs.

## High-level Summary

- A `RepoLink` model stores the association: ProjectName, DocumentationRepo (GitHub repo URL or path), CodebaseRepo (similar).
- The `RepoLinkService` handles creation and retrieval of these links.
- The core functionality is in `LinkRepositories(projectName, docRepo, codebaseRepo)`, which:
  1. Validates inputs are non-empty.
  2. Creates a `RepoLink` record in the database.
  3. Calls `FumadocsService.CreateFumadocsProject(docRepo)` to initialize a Fumadocs project from the documentation repo.
  4. Logs success or errors using Wails runtime logging.
- This allows the app to link code and docs, facilitating synced documentation generation or hosting.

## Backend (Go)

### Relevant Files
- `internal/models/repo_link.go`: Defines the `RepoLink` struct.
- `internal/repositories/repo_link_repository.go`: GORM-based CRUD for `RepoLink` (Create, FindByID, List with pagination).
- `internal/services/repo_link_service.go`: Service logic, including integration with `FumadocsService`.

### Model
```go
type RepoLink struct {
    ID                uint   `gorm:"primaryKey"`
    DocumentationRepo string
    CodebaseRepo      string
    ProjectName       string
}
```
Simple struct for persisting links.

### Repository
Implements `RepoLinkRepository` interface:
- `Create(ctx, link)`: Inserts via `db.Create()`.
- `FindByID(ctx, id)`: Retrieves via `db.First()`.
- `List(ctx, limit, offset)`: Paginates via `db.Limit().Offset().Find()`.

### Service
Implements `RepoLinkService`:
- `Register(projectName, docRepo, codeRepo)`: Validates, creates `RepoLink`, calls repo.Create, returns link or error.
- `Get(id)` / `List(limit, offset)`: Delegates to repository.
- `Startup(ctx)`: Stores context for logging.
- `LinkRepositories(...)`: 
  - Checks if service is available.
  - Calls `Register`.
  - On success, calls `fumadocsService.CreateFumadocsProject(docRepo)` (assumed to set up docs site).
  - Logs info: "Successfully linked project: {project}, doc: {doc} with codebase: {code}".
  - Errors are wrapped and logged (e.g., "failed to create fumadocs project").

## Integration with Fumadocs
- Fumadocs is likely a Next.js-based documentation framework.
- `CreateFumadocsProject` probably clones/pulls the doc repo and runs setup commands to generate a docs site linked to the codebase.
- The response from this call is logged for debugging.

## Usage Flow Example
1. User provides project name, doc repo (e.g., "user/docs-repo"), codebase repo (e.g., "user/code-repo").
2. App calls `repoLinkService.LinkRepositories(...)`.
3. Service registers in DB.
4. Service creates Fumadocs project from doc repo.
5. On success, link is established; app can now reference the linked repos for features like code-doc syncing or AI-assisted doc generation.

## Notes and Considerations
- No uniqueness constraints shown (e.g., multiple links per project?); DB schema may enforce.
- Errors from repo ops or Fumadocs are propagated with logging.
- Service depends on injected `FumadocsService`; ensure it's available.
- For production, add validation for repo URLs (e.g., GitHub format).
- Tests exist in `internal/tests/unit-tests/repo_link_service_test.go` covering registration errors.