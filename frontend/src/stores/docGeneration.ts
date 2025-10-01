import type { models } from "@go/models";
import { create } from "zustand";
import { type DemoEvent, demoEventSchema } from "@/types/events";
import {
	CommitDocs,
	GenerateDocs,
	StopStream,
} from "../../wailsjs/go/services/ClientService";
import { EventsOn } from "../../wailsjs/runtime";

export type DocGenerationStatus =
	| "idle"
	| "running"
	| "success"
	| "error"
	| "committing"
	| "canceled";

type StartArgs = {
	projectId: number;
	sourceBranch: string;
	targetBranch: string;
	provider: string;
};

type CommitArgs = {
	projectId: number;
	branch: string;
	files: string[];
};

type ProjectKey = string;

type DocGenerationData = {
	events: DemoEvent[];
	status: DocGenerationStatus;
	result: models.DocGenerationResult | null;
	error: string | null;
	cancellationRequested: boolean;
};

type State = {
	docStates: Record<ProjectKey, DocGenerationData>;
	start: (args: StartArgs) => Promise<void>;
	reset: (projectId: number | string) => void;
	commit: (args: CommitArgs) => Promise<void>;
	cancel: (projectId: number | string) => Promise<void>;
};

const EMPTY_DOC_STATE: DocGenerationData = {
	events: [],
	status: "idle",
	result: null,
	error: null,
	cancellationRequested: false,
};

const toKey = (projectId: number | string): ProjectKey => String(projectId);

const messageFromError = (error: unknown) => {
	if (error instanceof Error) {
		return error.message;
	}
	if (typeof error === "string") {
		return error;
	}
	return "An unknown error occurred while generating documentation.";
};

const createLocalEvent = (
	type: DemoEvent["type"],
	message: string
): DemoEvent => ({
	id:
		typeof crypto !== "undefined" && "randomUUID" in crypto
			? crypto.randomUUID()
			: Math.random().toString(36).slice(2),
	message,
	type,
	timestamp: new Date(),
});

type SubscriptionMap = {
	tool?: () => void;
	done?: () => void;
};

const subscriptions = new Map<ProjectKey, SubscriptionMap>();

const clearSubscriptions = (key: ProjectKey) => {
	const entry = subscriptions.get(key);
	if (!entry) {
		return;
	}
	entry.tool?.();
	entry.done?.();
	subscriptions.delete(key);
};

export const useDocGenerationStore = create<State>((set, get) => {
	const setDocState = (
		key: ProjectKey,
		partial: Partial<DocGenerationData> | ((prev: DocGenerationData) => DocGenerationData)
	) => {
		set((state) => {
			const previous = state.docStates[key] ?? EMPTY_DOC_STATE;
			const next =
				typeof partial === "function" ? partial(previous) : { ...previous, ...partial };
			return {
				docStates: {
					...state.docStates,
					[key]: next,
				},
			};
		});
	};

	return {
		docStates: {},

		start: async ({
			projectId,
			sourceBranch,
			targetBranch,
			provider,
		}: StartArgs) => {
			const key = toKey(projectId);
			const currentState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (currentState.status === "running") {
				return;
			}

			setDocState(key, {
				events: [],
				error: null,
				result: null,
				status: "running",
				cancellationRequested: false,
			});

			clearSubscriptions(key);

			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					setDocState(key, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation tool event", error, payload);
				}
			});

			const doneUnsub = EventsOn("events:llm:done", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					setDocState(key, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation done event", error, payload);
				}
			});

			subscriptions.set(key, { tool: toolUnsub, done: doneUnsub });

			try {
				const result = await GenerateDocs(
					projectId,
					sourceBranch,
					targetBranch,
					provider
				);
				setDocState(key, {
					result,
					status: "success",
					cancellationRequested: false,
				});
			} catch (error) {
				const message = messageFromError(error);
				const normalized = message.toLowerCase();
				const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
				const canceled =
					docState.cancellationRequested ||
					normalized.includes("context canceled") ||
					normalized.includes("context cancelled") ||
					normalized.includes("cancelled") ||
					normalized.includes("canceled");
				if (canceled) {
					setDocState(key, (prev) => ({
						...prev,
						error: null,
						result: null,
						status: "canceled",
						cancellationRequested: false,
						events: [
							...prev.events,
							createLocalEvent(
								"warn",
								"Documentation generation canceled by user."
							),
						],
					}));
				} else {
					setDocState(key, {
						error: message,
						status: "error",
						cancellationRequested: false,
						result: null,
					});
				}
			} finally {
				clearSubscriptions(key);
				setDocState(key, { cancellationRequested: false });
			}
		},

		cancel: async (projectId: number | string) => {
			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (docState.status !== "running") {
				return;
			}

			setDocState(key, { cancellationRequested: true });
			try {
				await StopStream();
			} catch (error) {
				const message = messageFromError(error);
				console.error("Failed to cancel doc generation", error);
				setDocState(key, (prev) => ({
					...prev,
					cancellationRequested: false,
					error: message,
					status: "error",
					result: null,
					events: [
						...prev.events,
						createLocalEvent(
							"error",
							`Failed to cancel documentation generation: ${message}`
						),
					],
				}));
			}
		},

		commit: async ({ projectId, branch, files }: CommitArgs) => {
			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (docState.status === "committing") {
				return;
			}

			setDocState(key, (prev) => ({
				...prev,
				error: null,
				status: "committing",
				events: [
					...prev.events,
					createLocalEvent(
						"info",
						`Committing documentation updates to ${branch}`
					),
				],
			}));

			try {
				await CommitDocs(projectId, branch, files);
				setDocState(key, (prev) => ({
					...prev,
					error: null,
					status: "success",
					events: [
						...prev.events,
						createLocalEvent(
							"info",
							`Committed documentation changes for ${branch}`
						),
					],
				}));
			} catch (error) {
				const message = messageFromError(error);
				setDocState(key, (prev) => ({
					...prev,
					error: message,
					status: "error",
					events: [
						...prev.events,
						createLocalEvent(
							"error",
							`Failed to commit documentation changes: ${message}`
						),
					],
				}));
			}
		},

		reset: (projectId: number | string) => {
			const key = toKey(projectId);
			clearSubscriptions(key);
			setDocState(key, {
				events: [],
				error: null,
				result: null,
				status: "idle",
				cancellationRequested: false,
			});
		},
	};
});
