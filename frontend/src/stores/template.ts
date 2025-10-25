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
			const defaultTemplates: models.Template[] = [
				{
					id: -1,
					name: "None",
					content: "",
				},
				{
					id: -2,
					name: "End user — Non technique",
					content:
						"Documentation destinée aux utilisateurs finaux. Expliquez la navigation de l'application, les fonctionnalités clés et fournissez des exemples d'utilisation pas à pas. " +
						"Incluez une section FAQ et des solutions aux problèmes courants pour aider les utilisateurs non techniques à accomplir leurs tâches sans jargon technique.",
				},
				{
					id: -3,
					name: "API",
					content:
						"Documentation d'API technique. Décrivez les endpoints, méthodes HTTP, schémas de requêtes et réponses, paramètres et en-têtes. " +
						"Fournissez des exemples de requêtes cURL/HTTP et des exemples de réponses JSON, ainsi que les codes d'erreur possibles et les bonnes pratiques d'authentification et de pagination.",
				},
				{
					id: -4,
					name: "Internal knowledge — Développeurs",
					content:
						"Documentation interne destinée aux développeurs travaillant sur le projet. " +
						"Couvrir l'architecture du projet, conventions de code, scripts de développement, procédure de build et déploiement, tests et intégration continue. " +
						"Inclure des notes sur les décisions techniques, les dépendances critiques et les points d'extension/maintenance.",
				},
			];

			set({ templates: defaultTemplates.concat(list), loading: false });
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
