import { create } from "zustand";
import { type DemoEvent, demoEventSchema } from "@/types/events";
import {
	ExploreDemo,
	StopStream,
} from "../../wailsjs/go/services/ClientService";
import { EventsOn } from "../../wailsjs/runtime";

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

		unsubscribeEvents = EventsOn("event:llm:tool", (payload) => {
			try {
				const evt = demoEventSchema.parse(payload);
				set((s) => ({ events: [...s.events, evt] }));
			} catch (error) {
				console.error("Invalid demo event payload:", error, payload);
			}
		});

		unsubscribeDone = EventsOn("events:llm:done", () => {
			set({ isListening: false });
			unsubscribeEvents?.();
			unsubscribeEvents = null;
			unsubscribeDone?.();
			unsubscribeDone = null;
		});

		set({ isListening: true });
		try {
			await ExploreDemo();
		} catch (e) {
			console.error("Failed to start demo event", e);
			set({ isListening: false });
		}
	},

	stop: async () => {
		try {
			await StopStream();
		} catch (e) {
			console.error("Failed to stop dem event", e);
		}
	},
}));
