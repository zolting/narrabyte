import { createRouter, RouterProvider } from "@tanstack/react-router";
import React from "react";
import { createRoot } from "react-dom/client";
import "./style.css";
import "./i18n";
import { routeTree } from "./routeTree.gen";
import { useAppSettingsStore } from "./stores/appSettings";

const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
	interface Register {
		router: typeof router;
	}
}

const container = document.getElementById("root") as HTMLElement;

const root = createRoot(container);

// Kick off AppSettings initialisation (non-blocking)
useAppSettingsStore.getState().init();

root.render(
	<React.StrictMode>
		<RouterProvider router={router} />
	</React.StrictMode>,
);
