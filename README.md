# Narrabyte

Narrabyte is a desktop app for generating, reviewing, and committing documentation updates from changes in a codebase. It links a source repository with a documentation repository, uses an LLM to draft documentation changes, and keeps the review flow close to Git: branch selection, generated diffs, refinement, commits, and merges.

The app is built with [Wails](https://wails.io/), a Go backend, and a React/TypeScript frontend.

## What Narrabyte does

- Links a code repository and documentation repository as a project.
- Generates documentation from a source branch, target branch, or single branch.
- Streams LLM tool activity so you can see what the generation agent is doing.
- Lets you review generated documentation in a diff view before accepting it.
- Supports follow-up refinement instructions on an existing documentation session.
- Commits generated docs to a documentation branch and can merge docs back into the source branch.
- Stores API keys in the operating system keyring.
- Supports OpenAI, Anthropic, and Google Gemini model providers.
- Provides reusable instruction templates for documentation runs.

## Install

Download the latest build from the [GitHub Releases page](https://github.com/zolting/narrabyte/releases).

Release builds are produced for:

- macOS
- Windows
- Ubuntu/Linux

After launching Narrabyte, open Settings and add an API key for the provider you want to use. API keys are stored through your operating system keyring under the `narrabyte` service name.

## Requirements for Using the App

You will need:

- Git repositories for both your codebase and your documentation.
- A clean enough working tree for Git operations that modify branches.
- An API key for at least one supported LLM provider.

Narrabyte works with local repositories. When you add a project, choose the documentation directory and the codebase directory on disk.

## Development Setup

### Prerequisites

Install:

- Go 1.25 or newer.
- Node.js 24 or newer.
- npm.
- Wails CLI v2.
- Git.

Install Wails with:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

On Linux, Wails also needs WebKit/GTK development packages. For Ubuntu:

```sh
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev libglib2.0-dev
```

### Clone and Install Dependencies

```sh
git clone git@github.com:zolting/narrabyte.git
cd narrabyte
cd frontend
npm install
cd ..
```

### Run in Development Mode

```sh
wails dev
```

This starts the Wails app with a Vite development server for frontend hot reload. Wails also exposes a browser development endpoint, usually at `http://localhost:34115`, where the frontend can call Go bindings while running in a browser.

In development builds, SQLite data is written to `narrabyte.db` in the project root. Production builds store data under the user's config directory in a `narrabyte` folder.

## Build

Build a production app locally:

```sh
wails build -tags prod
```

On Linux, use the WebKit 4.1 tag:

```sh
wails build -tags "prod,webkit2_41"
```

Build output is written under `build/bin`.

## Test and Validate

Run Go unit tests from the repository root:

```sh
go test ./internal/tests/unit-tests -v
```

Run Go integration tests:

```sh
go test ./internal/tests/integration-tests -v
```

The shell scripts in `scripts/` are also available, but they expect to be run from inside the `scripts` directory:

```sh
cd scripts
./run_unit_tests.sh
./run_integration_tests.sh
./run_all_tests.sh
```

Validate the frontend:

```sh
cd frontend
npm run build
npm run format
```

`npm run build` generates TanStack Router route files, type-checks the frontend, and builds the Vite bundle. `npm run format` runs Biome through Ultracite and writes formatting/lint fixes.

## Project Structure

```text
.
├── main.go                     # Wails app setup and service binding
├── app.go                      # App lifecycle and native file/directory dialogs
├── internal/
│   ├── database/               # SQLite setup and migrations
│   ├── events/                 # Runtime event emission for progress/tool logs
│   ├── llm/                    # LLM clients, prompts, and file-editing tools
│   ├── models/                 # Domain models and GORM entities
│   ├── repositories/           # Data access interfaces and GORM implementations
│   ├── services/               # Business logic exposed to Wails/frontend
│   └── tests/                  # Unit, integration, and mock packages
├── frontend/
│   ├── src/components/         # React UI components
│   ├── src/routes/             # TanStack Router routes
│   ├── src/stores/             # Zustand state stores
│   ├── src/assets/locales/     # English and French translations
│   └── wailsjs/                # Generated Wails bindings
├── scripts/                    # Test helper scripts
├── docs/                       # Internal implementation notes
└── wails.json                  # Wails project configuration
```

## Architecture Notes

The backend follows a layered structure:

- Wails bindings expose Go services to the frontend.
- `internal/services/` contains orchestration and application behavior.
- `internal/repositories/` handles database access behind interfaces.
- `internal/models/` contains persisted domain entities.
- `internal/llm/` wraps provider clients and the documentation-generation tools.

SQLite is used through GORM with WAL mode enabled. The app limits SQLite to one open connection to avoid local database lock contention.

The frontend uses React, TanStack Router, Zustand, i18next, Tailwind CSS, Radix UI primitives, and Biome/Ultracite for formatting and linting.

## Contributing

Contributions are welcome. Good first places to look are UI polish, documentation-generation workflow edge cases, provider/model support, tests around Git operations, and localization improvements.

Before opening a pull request:

1. Keep changes scoped and follow the existing service/component patterns.
2. Add or update tests when behavior changes.
3. Use translations for user-facing frontend text in both `frontend/src/assets/locales/en.json` and `frontend/src/assets/locales/fr.json`.
4. Avoid committing generated databases, local build outputs, API keys, or keyring/provider config.
5. Run the relevant Go tests.
6. Run `npm run build` in `frontend/` for frontend changes.

For frontend code, prefer the project's existing UI primitives and CSS variables. For backend code, wrap errors with context and keep Git/file operations scoped to the intended project directories.

## Releases

The release workflow runs when a tag matching `v*` is pushed. It builds Narrabyte for Ubuntu, Windows, and macOS, uploads artifacts, and creates a GitHub Release.

To publish a new release:

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

Replace `vX.Y.Z` with the version you want to publish.
