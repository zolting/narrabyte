import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { useEffect } from "react";
import { AppSidebar } from "@/components/Sidebar";
import {
	SidebarInset,
	SidebarProvider,
	SidebarTrigger,
} from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
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
			<Toaster />
			<ThemeSync />
			<AppSidebar />
			<SidebarInset className="flex flex-col">
				<header className="sticky top-0 z-10 flex h-16 shrink-0 items-center justify-between gap-2 border-b bg-background px-4">
					<SidebarTrigger className="-ml-1" />
				</header>
				<main>
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
