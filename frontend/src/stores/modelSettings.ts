import type { models } from "@go/models";
import {
	GetModel,
	ListModelGroups,
	SetModelEnabled,
	SetProviderEnabled,
} from "@go/services/modelConfigService";
import { create } from "zustand";

export type ModelKey = string;

export type ModelOption = {
	key: string;
	displayName: string;
	apiName: string;
	providerId: string;
	providerName: string;
	reasoningEffort?: string;
	thinking?: boolean | null;
	enabled: boolean;
};

export type ModelGroup = {
	providerId: string;
	providerName: string;
	models: ModelOption[];
};

type State = {
	groups: ModelGroup[];
	initialized: boolean;
	loading: boolean;
	error: string | null;
	defaultModelKey: ModelKey | null;
	setDefaultModelKey: (modelKey: ModelKey) => void;
	init: () => Promise<void>;
	toggleModel: (modelKey: string, enabled: boolean) => Promise<void>;
	toggleProvider: (providerId: string, enabled: boolean) => Promise<void>;
	reloadModel: (modelKey: string) => Promise<void>;
};

const mapModel = (model: models.LLMModel): ModelOption => ({
	key: model.key,
	displayName: model.displayName,
	apiName: model.apiName,
	providerId: model.providerId,
	providerName: model.providerName,
	reasoningEffort: model.reasoningEffort || undefined,
	thinking: typeof model.thinking === "boolean" ? model.thinking : null,
	enabled: Boolean(model.enabled),
});

const mapGroup = (group: models.LLMModelGroup): ModelGroup => ({
	providerId: group.providerId,
	providerName: group.providerName,
	models: (group.models ?? []).map(mapModel),
});

export const useModelSettingsStore = create<State>((set, get) => ({
	groups: [],
	initialized: false,
	loading: false,
	error: null,
	defaultModelKey: null,

	setDefaultModelKey: (modelKey: ModelKey) =>
		set({ defaultModelKey: modelKey }),

	init: async () => {
		if (get().loading) {
			return;
		}
		set({ loading: true, error: null });
		try {
			const result = await ListModelGroups();
			set({
				groups: Array.isArray(result) ? result.map(mapGroup) : [],
				initialized: true,
				loading: false,
			});
		} catch (err) {
			set({
				error: err instanceof Error ? err.message : String(err),
				loading: false,
				initialized: true,
				groups: [],
			});
		}
	},

	toggleModel: async (modelKey, enabled) => {
		try {
			const updated = await SetModelEnabled(modelKey, enabled);
			set((state) => ({
				groups: state.groups.map((group) => ({
					...group,
					models: group.models.map((model) =>
						model.key === modelKey ? mapModel(updated) : model
					),
				})),
				error: null,
			}));
		} catch (err) {
			set({ error: err instanceof Error ? err.message : String(err) });
		}
	},

	toggleProvider: async (providerId, enabled) => {
		try {
			const updatedModels = await SetProviderEnabled(providerId, enabled);
			const mapped = Array.isArray(updatedModels)
				? updatedModels.map(mapModel)
				: [];
			set((state) => ({
				groups: state.groups.map((group) =>
					group.providerId === providerId
						? {
								...group,
								models: group.models.map((model) => {
									const next = mapped.find((m) => m.key === model.key);
									return next ? next : { ...model, enabled };
								}),
							}
						: group
				),
				error: null,
			}));
		} catch (err) {
			set({ error: err instanceof Error ? err.message : String(err) });
		}
	},

	reloadModel: async (modelKey) => {
		try {
			const refreshed = await GetModel(modelKey);
			set((state) => ({
				groups: state.groups.map((group) => ({
					...group,
					models: group.models.map((model) =>
						model.key === modelKey ? mapModel(refreshed) : model
					),
				})),
			}));
		} catch {
			// ignore reload errors - caller can refetch via init
		}
	},
}));
