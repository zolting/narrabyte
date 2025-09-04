import { create } from "zustand";
import { StartDemoEvents, StopDemoEvents } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { type DemoEvent, demoEventSchema } from "../types/events";

type State = {
	events: DemoEvent[];
	isListening: boolean;
	start: () => Promise<void>;
	stop: () => Promise<void>;
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

		unsubscribeEvents = EventsOn("events:demo", (payload) => {
			try {
				// Validate and parse the incoming payload with Zod
				const evt = demoEventSchema.parse(payload);
				set((s) => ({ events: [...s.events, evt] }));
			} catch (error) {
				// Handle validation errors gracefully
				console.error("Invalid demo event payload:", error, payload);
				// Optionally emit an error event or show user notification
			}
		});

		unsubscribeDone = EventsOn("events:demo:done", () => {
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
			console.error("Failed to start demo event", e);
			set({ isListening: false });
		}
	},

	stop: async () => {
		try {
			await StopDemoEvents();
		} catch (e) {
			console.error("Failed to stop dem event", e);
		}
	},
}));
