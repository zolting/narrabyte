import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { useEffect } from "react";
import { AppSidebar } from "@/components/Sidebar";
import {
	SidebarInset,
	SidebarProvider,
	SidebarTrigger,
} from "@/components/ui/sidebar";
import { useAppSettingsStore } from "@/stores/appSettings";

function ThemeSync() {
	const { settings } = useAppSettingsStore();

	useEffect(() => {
		const root = window.document.documentElement;
		const appTheme = (settings?.Theme ?? "system") as
			| "light"
			| "dark"
			| "system";

		const apply = (mode: "light" | "dark") => {
			root.classList.remove("light", "dark");
			root.classList.add(mode);
		};

		const mql = window.matchMedia("(prefers-color-scheme: dark)");
		const resolveSystem = () => (mql.matches ? "dark" : "light");

		if (appTheme === "system") {
			apply(resolveSystem());
		} else {
			apply(appTheme);
		}

		const onChange = () => {
			if (appTheme === "system") {
				apply(resolveSystem());
			}
		};

		mql.addEventListener("change", onChange);
		return () => {
			mql.removeEventListener("change", onChange);
		};
	}, [settings?.Theme]);

	return null;
}

function RootLayout() {
	return (
		<SidebarProvider>
			<ThemeSync />
			<AppSidebar />
			<SidebarInset>
				<header className="flex h-16 shrink-0 items-center gap-2 border-b px-4">
					<SidebarTrigger className="-ml-1" />
				</header>
				<main className="flex-1 overflow-auto p-6">
					<Outlet />
					<TanStackRouterDevtools position="bottom-right" />
				</main>
			</SidebarInset>
		</SidebarProvider>
	);
}

export const Route = createRootRoute({
	component: RootLayout,
});
