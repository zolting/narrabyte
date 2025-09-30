import { createFileRoute } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import AddApiKeyDialog from "@/components/AddApiKeyDialog";
import ApiKeyManager from "@/components/ApiKeyManager";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { type AppTheme, useAppSettingsStore } from "../stores/appSettings";

export const Route = createFileRoute("/settings")({
	component: Settings,
});

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
						<CardTitle>{t("settings.preferences")}</CardTitle>
						<CardDescription>
							{t("settings.preferencesDescription")}
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<div className="flex items-center justify-between">
							<span className="font-medium text-sm">{t("settings.theme")}</span>
							<Select
								disabled={isLoading}
								onValueChange={(value) => setTheme(value as AppTheme)}
								value={theme}
							>
								<SelectTrigger className="w-[180px]">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="light">{t("settings.light")}</SelectItem>
									<SelectItem value="dark">{t("settings.dark")}</SelectItem>
									<SelectItem value="system">{t("settings.system")}</SelectItem>
								</SelectContent>
							</Select>
						</div>

						<div className="flex items-center justify-between">
							<span className="font-medium text-sm">
								{t("settings.language")}
							</span>
							<Select
								disabled={isLoading}
								onValueChange={setLocale}
								value={locale}
							>
								<SelectTrigger className="w-[180px]">
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="en">English</SelectItem>
									<SelectItem value="fr">Français</SelectItem>
								</SelectContent>
							</Select>
						</div>
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
					<ArrowLeft size={16} />
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
