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
- If more locales will be added, keep normalizeToSupportedLocale in sync with i18n supported languages.

## Git Diff frontend component

This project includes a small Git diff viewer used in the UI to show changes in a familiar unified/split view. The implementation is a lightweight dialog wrapper around the react-diff-view library and a small theme override. Key facts below are directly verifiable in the source.

Summary

- The component uses react-diff-view to parse and render git patch text into file/hunk views.
- It currently contains a SAMPLE_DIFF constant used as placeholder content; the parsed result is memoized with useMemo to avoid repeated parsing.
- The dialog exposes a view toggle (split vs unified) and renders each hunk with the Hunk component from react-diff-view.
- Styling is provided by a local CSS theme file which sets react-diff-view CSS variables and layout rules.

How it works (evidence-backed)

- Parsing and rendering
  - The component imports parseDiff, Diff and Hunk from react-diff-view and calls parseDiff(SAMPLE_DIFF) inside useMemo to produce a `files` array (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L1-L4, #L43-L46, #L15-L33).
  - It selects the first parsed file and renders a <Diff> component with diffType and thunks props, then maps each hunk to a <Hunk /> element (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L78-L90).

- View mode toggle
  - The component keeps local state viewType ("split" | "unified") and toggles it when the user clicks the button; this value is passed to the <Diff> component as `viewType` (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L41-L51, #L84-L85).

- Placeholder content and memoization
  - A SAMPLE_DIFF constant in the file contains example patch text; because parseDiff can be expensive, the code memoizes parsing with useMemo (source: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L15-L33, #L43-L46).

- Styling
  - The component imports a local CSS file diff-view-theme.css which defines CSS variables used by react-diff-view (e.g. --diff-background-color, --diff-text-color) and layout rules to make the diff table full-width (source: frontend/src/components/GitDiffDialog/diff-view-theme.css#L1-L6, #L52-L56).

Dependencies

- The project depends on react-diff-view (see frontend/package.json dependencies) which provides the parseDiff/Diff/Hunk primitives used by the component (source: frontend/package.json#L29-L29).

Inferred / Notes

- Inferred: The current GitDiffDialog is implemented as a self-contained dialog with placeholder diff text (SAMPLE_DIFF). That implies it is intended as a reusable UI building block; to display a real git diff the component would need to accept the patch text as a prop and call parseDiff on that string instead of SAMPLE_DIFF (Inference based on the presence of SAMPLE_DIFF and the lack of a prop for diff content in the current file: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx#L15-L33).

- Inferred: The component does not fetch diffs itself (no network/git calls inside the component). Backend or higher-level UI code should supply the raw git patch string when integrating this dialog into a flows that calculate diffs (no code for fetching diffs found in this component; see git service in internal/services/git_service.go for server-side diff generation: internal/services/git_service.go#L96-L96).

Where to look in code (sources)

- Component implementation: frontend/src/components/GitDiffDialog/GitDiffDialog.tsx (imports, SAMPLE_DIFF, parseDiff usage, view toggle, rendering) — see lines: #L1-L4, #L15-L33, #L41-L51, #L43-L46, #L78-L90.
- Theme / style overrides: frontend/src/components/GitDiffDialog/diff-view-theme.css (CSS variables and layout rules) — see lines: #L1-L6, #L52-L56.
- Dependency: frontend/package.json (react-diff-view entry) — see line: #L29-L29.
- Server-side diff generation (integration point): internal/services/git_service.go (DiffBetweenCommits) — see line: #L96-L96.

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