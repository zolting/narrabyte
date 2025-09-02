import { create } from "zustand";
import { StartDemoEvents } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

export type DemoEvent = {
	id: number;
	type: "info" | "debug" | "warn" | "error" | string;
	message: string;
	timestamp: string;
};

type State = {
	events: DemoEvent[];
	isListening: boolean;
	start: () => Promise<void>;
	clear: () => void;
};

let unsubscribeEvents: (() => void) | null = null;
let unsubscribeDone: (() => void) | null = null;

export const useDemoEventsStore = create<State>((set, get) => ({
	events: [],
	isListening: false,

	clear: () => set({ events: [] }),

	start: async () => {
		if (get().isListening) {
			return;
		}

		// Subscribe to demo events
		unsubscribeEvents?.();
		unsubscribeDone?.();

		unsubscribeEvents = EventsOn("demo:events", (payload) => {
			// Payload comes from Go struct; convert to our TS shape
			const evt: DemoEvent = {
				id: payload?.id ?? 0,
				type: payload?.type ?? "info",
				message: payload?.message ?? "",
				timestamp: payload?.timestamp ?? new Date().toISOString(),
			};
			set((s) => ({ events: [...s.events, evt] }));
		});

		unsubscribeDone = EventsOn("demo:events:done", () => {
			set({ isListening: false });
			unsubscribeEvents?.();
			unsubscribeEvents = null;
			unsubscribeDone?.();
			unsubscribeDone = null;
		});

		set({ isListening: true });
		try {
			await StartDemoEvents();
		} catch (e) {
			console.error("Failed to start demo events", e);
			set({ isListening: false });
		}
	},
}));
