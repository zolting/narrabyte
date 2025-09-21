import type { models } from "@go/models";
import { create } from "zustand";
import { type DemoEvent, demoEventSchema } from "@/types/events";
import {
	CommitDocs,
	GenerateDocs,
} from "../../wailsjs/go/services/ClientService";
import { EventsOn } from "../../wailsjs/runtime";

export type DocGenerationStatus =
	| "idle"
	| "running"
	| "success"
	| "error"
	| "committing";

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
	start: (args: StartArgs) => Promise<void>;
	reset: () => void;
	commit: (args: CommitArgs) => Promise<void>;
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

	start: async ({ projectId, sourceBranch, targetBranch }: StartArgs) => {
		if (get().status === "running") {
			return;
		}

		set({
			events: [],
			error: null,
			result: null,
			status: "running",
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
			set({ result, status: "success" });
		} catch (error) {
			set({ error: messageFromError(error), status: "error" });
		} finally {
			unsubscribeTool?.();
			unsubscribeTool = null;
			unsubscribeDone?.();
			unsubscribeDone = null;
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
		});
	},
}));
