import {
	createRootRoute,
	ErrorComponent,
	Outlet,
} from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { useEffect } from "react";
import { CurrentGenerationsIndicator } from "@/components/CurrentGenerationsIndicator";
import {
	ProjectCacheHost,
	ProjectCacheProvider,
} from "@/components/projects/ProjectCache";
import { AppSidebar } from "@/components/Sidebar";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
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
		<ProjectCacheProvider>
			<SidebarProvider className="h-screen w-full overflow-hidden">
				<Toaster />
				<ThemeSync />
				<AppSidebar />
				<SidebarInset className="flex h-full w-full flex-col overflow-hidden">
					<div className="flex w-full justify-end px-4 py-2">
						<CurrentGenerationsIndicator />
					</div>
					<main className="flex min-h-0 flex-1 flex-col overflow-hidden">
						<ProjectCacheHost />
						<Outlet />
							{process.env.NODE_ENV === "development" && (
		<TanStackRouterDevtools position="bottom-right" />
	)}
					</main>
				</SidebarInset>
			</SidebarProvider>
		</ProjectCacheProvider>
	);
}

export const Route = createRootRoute({
	component: RootLayout,
	errorComponent: ({ error }) => <ErrorComponent error={error} />,
});
