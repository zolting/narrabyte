import { createFileRoute } from "@tanstack/react-router";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import AddApiKeyDialog from "@/components/AddApiKeyDialog";
import ApiKeyManager from "@/components/ApiKeyManager";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAppSettingsStore } from "../stores/appSettings";

export const Route = createFileRoute("/settings")({
	component: Settings,
});

function ThemeSelector({
	theme,
	setTheme,
	loading,
}: {
	theme: string;
	setTheme: (theme: string) => void;
	loading: boolean;
}) {
	const { t } = useTranslation();

	if (loading) {
		return (
			<div className="text-center text-muted-foreground text-sm">
				{t("settings.loading")}
			</div>
		);
	}

	return (
		<div className="flex gap-2">
			<Button
				onClick={() => setTheme("light")}
				variant={theme === "light" ? "default" : "outline"}
			>
				{t("settings.light")}
			</Button>
			<Button
				onClick={() => setTheme("dark")}
				variant={theme === "dark" ? "default" : "outline"}
			>
				{t("settings.dark")}
			</Button>
			<Button
				onClick={() => setTheme("system")}
				variant={theme === "system" ? "default" : "outline"}
			>
				{t("settings.system")}
			</Button>
		</div>
	);
}

function LanguageSelector({
	locale,
	setLocale,
	loading,
}: {
	locale: string;
	setLocale: (locale: string) => void;
	loading: boolean;
}) {
	const { t } = useTranslation();

	if (loading) {
		return (
			<div className="text-center text-muted-foreground text-sm">
				{t("settings.loading")}
			</div>
		);
	}

	return (
		<div className="flex gap-2">
			<Button
				onClick={() => setLocale("en")}
				variant={locale.startsWith("en") ? "default" : "outline"}
			>
				English
			</Button>
			<Button
				onClick={() => setLocale("fr")}
				variant={locale.startsWith("fr") ? "default" : "outline"}
			>
				Français
			</Button>
		</div>
	);
}

function Settings() {
	const { t } = useTranslation();
	const { settings, initialized, loading, setTheme, setLocale } =
		useAppSettingsStore();

	const [dialogOpen, setDialogOpen] = useState(false);
	const [editingProvider, setEditingProvider] = useState<string | undefined>(
		undefined
	);
	const apiKeyManagerRef = useRef<{ refresh: () => void }>(null);
	const theme = settings?.Theme ?? "system";
	const locale = settings?.Locale ?? "en";
	const isLoading = !initialized || loading;

	const handleKeyAdded = () => {
		apiKeyManagerRef.current?.refresh();
	};

	const handleAddClick = () => {
		setEditingProvider(undefined);
		setDialogOpen(true);
	};

	const handleEditClick = (provider: string) => {
		setEditingProvider(provider);
		setDialogOpen(true);
	};

	const handleCloseDialog = () => {
		setDialogOpen(false);
		setEditingProvider(undefined);
	};

	return (
		<div className="flex min-h-screen flex-col items-center bg-background p-8 font-mono">
			<div className="w-full max-w-2xl space-y-6">
				<h1 className="text-center font-bold text-2xl">
					{t("settings.title")}
				</h1>

				<Card>
					<CardHeader>
						<CardTitle>{t("settings.theme")}</CardTitle>
					</CardHeader>
					<CardContent>
						<ThemeSelector
							loading={isLoading}
							setTheme={setTheme}
							theme={theme}
						/>
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<CardTitle>{t("settings.language")}</CardTitle>
					</CardHeader>
					<CardContent>
						<LanguageSelector
							loading={isLoading}
							locale={locale}
							setLocale={setLocale}
						/>
					</CardContent>
				</Card>

				<ApiKeyManager
					onAddClick={handleAddClick}
					onEditClick={handleEditClick}
					ref={apiKeyManagerRef}
				/>

				<Button
					className="w-full"
					onClick={() => window.history.back()}
					variant="outline"
				>
					{t("common.goBack")}
				</Button>

				<AddApiKeyDialog
					editProvider={editingProvider}
					onClose={handleCloseDialog}
					onKeyAdded={handleKeyAdded}
					open={dialogOpen}
				/>
			</div>
		</div>
	);
}
