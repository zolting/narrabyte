import type { models } from "@go/models";
import { create } from "zustand";
import { type DemoEvent, demoEventSchema } from "@/types/events";
import { GenerateDocs } from "../../wailsjs/go/services/ClientService";
import { EventsOn } from "../../wailsjs/runtime";

export type DocGenerationStatus = "idle" | "running" | "success" | "error";

type StartArgs = {
	projectId: number;
	sourceBranch: string;
	targetBranch: string;
};

type State = {
	events: DemoEvent[];
	status: DocGenerationStatus;
	result: models.DocGenerationResult | null;
	error: string | null;
	start: (args: StartArgs) => Promise<void>;
	reset: () => void;
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
