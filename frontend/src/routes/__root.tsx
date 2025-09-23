import { createRootRoute, Link, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { GitBranch, Settings } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { GitDiffDialog } from "@/components/GitDiffDialog/GitDiffDialog";
import { AppSidebar } from "@/components/Sidebar";
import { Button } from "@/components/ui/button";
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
	const { t } = useTranslation();

	return (
		<SidebarProvider>
			<Toaster />
			<ThemeSync />
			<AppSidebar />
			<SidebarInset className="flex flex-col">
				<header className="sticky top-0 z-10 flex h-16 shrink-0 items-center justify-between gap-2 border-b bg-background px-4">
					<SidebarTrigger className="-ml-1" />
					<div className="flex gap-2">
						<GitDiffDialog>
							<Button className="text-foreground" size="icon" variant="outline">
								<GitBranch className="h-4 w-4 text-foreground" />
								<span className="sr-only">{t("common.viewDiff")}</span>
							</Button>
						</GitDiffDialog>
						<Button
							asChild
							className="text-foreground"
							size="icon"
							variant="outline"
						>
							<Link to="/settings">
								<Settings className="h-4 w-4 text-foreground" />
								<span className="sr-only">{t("common.settings")}</span>
							</Link>
						</Button>
					</div>
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
