import type { models } from "@go/models";
import { create } from "zustand";
import { type DemoEvent, demoEventSchema } from "@/types/events";
import {
	CommitDocs,
	GenerateDocs,
	RefineDocs,
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
};

type CommitArgs = {
	projectId: number;
	branch: string;
	files: string[];
};

type State = {
	events: DemoEvent[];
	status: DocGenerationStatus;
	result: models.DocGenerationResult | null;
	error: string | null;
	cancellationRequested: boolean;
	toggleChat: (open?: boolean) => void;
	chatOpen: boolean;
	messages: {
		id: string;
		role: "user" | "assistant";
		content: string;
		status?: "pending" | "sent" | "error";
		createdAt: Date;
	}[];
	start: (args: StartArgs) => Promise<void>;
	reset: () => void;
	commit: (args: CommitArgs) => Promise<void>;
	cancel: () => Promise<void>;
	refine: (args: {
		projectId: number;
		branch: string;
		instruction: string;
	}) => Promise<void>;
};

let unsubscribeTool: (() => void) | null = null;
let unsubscribeDone: (() => void) | null = null;

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

export const useDocGenerationStore = create<State>((set, get) => ({
	events: [],
	status: "idle",
	result: null,
	error: null,
	cancellationRequested: false,
	chatOpen: false,
	messages: [],

	start: async ({ projectId, sourceBranch, targetBranch }: StartArgs) => {
		if (get().status === "running") {
			return;
		}

		set({
			events: [],
			error: null,
			result: null,
			status: "running",
			cancellationRequested: false,
		});

		unsubscribeTool?.();
		unsubscribeDone?.();

		unsubscribeTool = EventsOn("event:llm:tool", (payload) => {
			try {
				const evt = demoEventSchema.parse(payload);
				set((state) => ({ events: [...state.events, evt] }));
			} catch (error) {
				console.error("Invalid doc generation tool event", error, payload);
			}
		});

		unsubscribeDone = EventsOn("events:llm:done", (payload) => {
			try {
				const evt = demoEventSchema.parse(payload);
				set((state) => ({ events: [...state.events, evt] }));
			} catch (error) {
				console.error("Invalid doc generation done event", error, payload);
			}
		});

		try {
			const result = await GenerateDocs(projectId, sourceBranch, targetBranch);
			set({ result, status: "success", cancellationRequested: false });
		} catch (error) {
			const message = messageFromError(error);
			const normalized = message.toLowerCase();
			const canceled =
				get().cancellationRequested ||
				normalized.includes("context canceled") ||
				normalized.includes("context cancelled") ||
				normalized.includes("cancelled") ||
				normalized.includes("canceled");
			if (canceled) {
				set((state) => ({
					error: null,
					result: null,
					status: "canceled",
					cancellationRequested: false,
					events: [
						...state.events,
						createLocalEvent(
							"warn",
							"Documentation generation canceled by user."
						),
					],
				}));
			} else {
				set({
					error: message,
					status: "error",
					cancellationRequested: false,
					result: null,
				});
			}
		} finally {
			unsubscribeTool?.();
			unsubscribeTool = null;
			unsubscribeDone?.();
			unsubscribeDone = null;
			set({ cancellationRequested: false });
		}
	},

	cancel: async () => {
		if (get().status !== "running") {
			return;
		}

		set({ cancellationRequested: true });
		try {
			await StopStream();
		} catch (error) {
			const message = messageFromError(error);
			console.error("Failed to cancel doc generation", error);
			set((state) => ({
				cancellationRequested: false,
				error: message,
				status: "error",
				result: null,
				events: [
					...state.events,
					createLocalEvent(
						"error",
						`Failed to cancel documentation generation: ${message}`
					),
				],
			}));
		}
	},

	commit: async ({ projectId, branch, files }: CommitArgs) => {
		if (get().status === "committing") {
			return;
		}

		set((state) => ({
			error: null,
			status: "committing",
			events: [
				...state.events,
				createLocalEvent(
					"info",
					`Committing documentation updates to ${branch}`
				),
			],
		}));

		try {
			await CommitDocs(projectId, branch, files);
			set((state) => ({
				error: null,
				status: "success",
				events: [
					...state.events,
					createLocalEvent(
						"info",
						`Committed documentation changes for ${branch}`
					),
				],
				result: state.result,
			}));
		} catch (error) {
			set((state) => ({
				error: messageFromError(error),
				status: "error",
				events: [
					...state.events,
					createLocalEvent(
						"error",
						`Failed to commit documentation changes: ${messageFromError(error)}`
					),
				],
			}));
		}
	},

	reset: () => {
		unsubscribeTool?.();
		unsubscribeTool = null;
		unsubscribeDone?.();
		unsubscribeDone = null;
		set({
			events: [],
			error: null,
			result: null,
			status: "idle",
			cancellationRequested: false,
			messages: [],
			chatOpen: false,
		});
	},

	toggleChat: (open) => {
		if (typeof open === "boolean") {
			set({ chatOpen: open });
		} else {
			set((s) => ({ chatOpen: !s.chatOpen }));
		}
	},

	refine: async ({ projectId, branch, instruction }) => {
		if (get().status === "running" || !instruction.trim()) {
			return;
		}

		// Optimistically render the user message
		const id =
			typeof crypto !== "undefined" && "randomUUID" in crypto
				? crypto.randomUUID()
				: Math.random().toString(36).slice(2);
		const userMsg = {
			id,
			role: "user" as const,
			content: instruction,
			status: "pending" as const,
			createdAt: new Date(),
		};

		set((s) => ({
			messages: [...s.messages, userMsg],
			events: s.events,
			error: null,
			status: "running",
		}));

		// Subscribe to events
		unsubscribeTool?.();
		unsubscribeDone?.();
		unsubscribeTool = EventsOn("event:llm:tool", (payload) => {
			try {
				const evt = demoEventSchema.parse(payload);
				set((state) => ({ events: [...state.events, evt] }));
			} catch (error) {
				console.error("Invalid refine tool event", error, payload);
			}
		});
		unsubscribeDone = EventsOn("events:llm:done", (payload) => {
			try {
				const evt = demoEventSchema.parse(payload);
				set((state) => ({ events: [...state.events, evt] }));
			} catch (error) {
				console.error("Invalid refine done event", error, payload);
			}
		});

		try {
			const result = await RefineDocs(projectId, branch, instruction);
			// Mark user message as sent and add assistant summary
			type ChatMessage = State["messages"][number];

			set((s) => {
				const updated: ChatMessage[] = s.messages.map(
					(m): ChatMessage =>
						m.id === id ? { ...m, status: "sent" as const } : m
				);

				const summary = (result?.summary ?? "").trim();
				const maybeAssistant: ChatMessage[] = summary
					? [
							{
								id:
									globalThis.crypto && "randomUUID" in globalThis.crypto
										? globalThis.crypto.randomUUID()
										: Math.random().toString(36).slice(2),
								role: "assistant" as const,
								content: summary,
								createdAt: new Date(),
							},
						]
					: [];

				return {
					messages: [...updated, ...maybeAssistant],
					result,
					status: "success",
					events: [
						...s.events,
						createLocalEvent(
							"info",
							"Applied user instruction to documentation."
						),
					],
				} satisfies Partial<State>;
			});
		} catch (error) {
			const message = messageFromError(error);
			set((s) => ({
				messages: s.messages
					.map((m) => (m.id === id ? { ...m, status: "error" as const } : m))
					.concat([
						{
							id: Math.random().toString(36).slice(2),
							role: "assistant" as const,
							content: `Error: ${message}`,
							createdAt: new Date(),
						},
					]),
				status: "error",
				error: message,
			}));
		} finally {
			unsubscribeTool?.();
			unsubscribeTool = null;
			unsubscribeDone?.();
			unsubscribeDone = null;
		}
	},
}));
