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

