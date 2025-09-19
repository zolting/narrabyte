import { create } from "zustand";
import "../i18n";
import { Get, Update } from "@go/services/appSettingsService";
import i18n from "i18next";

export type AppTheme = "light" | "dark" | "system";

export type AppSettings = {
	ID: number;
	Version: number;
	Theme: string;
	Locale: string;
	UpdatedAt?: string; // ISO string; may be zero-time when not persisted yet
};

type State = {
	settings: AppSettings | null;
	initialized: boolean;
	loading: boolean;
	error?: string;
	init: () => Promise<void>;
	setTheme: (theme: AppTheme) => Promise<void>;
	setLocale: (locale: string) => Promise<void>;
	update: (theme: AppTheme, locale: string) => Promise<void>;
};

function isZeroTimeISOString(value?: string): boolean {
	if (!value || typeof value !== "string") {
		return true;
	}
	// Go's zero time commonly serializes as "0001-01-01T00:00:00Z"
	if (value.startsWith("0001-01-01")) {
		return true;
	}
	const t = Date.parse(value);
	return Number.isNaN(t) || t <= 0;
}

function normalizeToSupportedLocale(input: string | undefined): string {
	const raw = (input || "en").toLowerCase();
	const base = raw.split("-")[0];
	// Keep in sync with i18n supportedLngs
	return base === "fr" ? "fr" : "en";
}

async function handleFirstRun(
	settings: AppSettings,
	set: (state: Partial<State>) => void
) {
	const detected = normalizeToSupportedLocale(
		i18n.language || navigator.language
	);
	const updated = await Update(settings.Theme ?? "system", detected);
	if (i18n.language !== detected) {
		i18n.changeLanguage(detected);
	}
	set({ settings: updated, initialized: true, loading: false });
}

function handleExistingSettings(
	settings: AppSettings,
	set: (state: Partial<State>) => void
) {
	const persisted = normalizeToSupportedLocale(settings.Locale);
	if (i18n.language !== persisted) {
		i18n.changeLanguage(persisted);
	}
	set({ settings, initialized: true, loading: false });
}

export const useAppSettingsStore = create<State>((set, get) => ({
	settings: null,
	initialized: false,
	loading: false,

	init: async () => {
		if (get().initialized) {
			return;
		}

		set({ loading: true, error: undefined });

		try {
			const settings = await Get();

			if (isZeroTimeISOString(settings.UpdatedAt)) {
				await handleFirstRun(settings, set);
			} else {
				handleExistingSettings(settings, set);
			}
		} catch (e: unknown) {
			set({
				error: e instanceof Error ? e.message : String(e),
				loading: false,
				initialized: true,
			});
		}
	},

	setTheme: async (theme: AppTheme) => {
		const current = get().settings;
		const locale = normalizeToSupportedLocale(current?.Locale || i18n.language);
		const updated = await Update(theme, locale);
		set({ settings: updated });
	},

	setLocale: async (locale) => {
		const current = get().settings;
		const theme = (current?.Theme ?? "system") as AppTheme;
		const normalized = normalizeToSupportedLocale(locale);
		const updated = await Update(theme, normalized);
		set({ settings: updated });
		if (i18n.language !== normalized) {
			i18n.changeLanguage(normalized);
		}
	},

	update: async (theme, locale) => {
		const normalized = normalizeToSupportedLocale(locale);
		const updated = await Update(theme, normalized);
		set({ settings: updated });
		if (i18n.language !== normalized) {
			i18n.changeLanguage(normalized);
		}
	},
}));
