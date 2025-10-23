import type { models } from "@go/models";
import {
	CreateTemplate,
	DeleteTemplate,
	ListTemplates,
	UpdateTemplate,
} from "@go/services/templateService";
import { create } from "zustand";

type TemplateState = {
	templates: models.Template[];
	currentTemplate: Partial<models.Template> | null;
	loading: boolean;
	error: string | null;
	loadTemplates: () => Promise<void>;
	createTemplate: (template: Partial<models.Template>) => Promise<void>;
	deleteTemplate: (id: number) => Promise<void>;
	editTemplate: (updatedTemplate: Partial<models.Template>) => Promise<void>;
};

export const useTemplateStore = create<TemplateState>((set, get) => ({
	templates: [],
	currentTemplate: null,
	loading: false,
	error: null,

	loadTemplates: async () => {
		set({ loading: true, error: null });
		try {
			const list = await ListTemplates();
			set({ templates: list, loading: false });
		} catch (err) {
			set({
				error: err instanceof Error ? err.message : String(err),
				loading: false,
			});
		}
	},

	createTemplate: async (template) => {
		set({ loading: true, error: null });
		try {
			if (!template.name || template.name.trim() === "") {
				throw new Error("template `name` is required");
			}
			if (!template.content || template.content.trim() === "") {
				throw new Error("template `content` is required");
			}
			const createTemplate = template as models.Template;
			await CreateTemplate(createTemplate);
			await get().loadTemplates();
		} catch (err) {
			set({ error: err instanceof Error ? err.message : String(err) });
		} finally {
			set({ loading: false });
		}
	},

	deleteTemplate: async (id) => {
		set({ loading: true, error: null });
		try {
			await DeleteTemplate(id);
			await get().loadTemplates();
		} catch (err) {
			set({ error: err instanceof Error ? err.message : String(err) });
		} finally {
			set({ loading: false });
		}
	},

	editTemplate: async (template) => {
		set({ loading: true, error: null });
		try {
			if (!template.name || template.name.trim() === "") {
				throw new Error("template `name` is required");
			}
			if (!template.content || template.content.trim() === "") {
				throw new Error("template `content` is required");
			}

			const updatedTemplate = template as models.Template;
			await UpdateTemplate(updatedTemplate);
			await get().loadTemplates();
		} catch (err) {
			set({ error: err instanceof Error ? err.message : String(err) });
		} finally {
			set({ loading: false });
		}
	},
}));
