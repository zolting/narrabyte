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
import { type AppTheme, useAppSettingsStore } from "@/stores/appSettings";

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
		<div className="min-h-screen w-full overflow-x-hidden bg-background font-mono">
			<div className="mx-auto w-full max-w-4xl px-6 py-12">
				<div className="mb-8">
					<Button
						className="mb-6"
						onClick={() => window.history.back()}
						size="sm"
						variant="ghost"
					>
						<ArrowLeft className="mr-2 h-4 w-4" />
						{t("common.goBack")}
					</Button>
					<h1 className="font-bold text-4xl tracking-tight">
						{t("settings.title")}
					</h1>
				</div>

				<div className="space-y-8">
					<Card>
						<CardHeader>
							<CardTitle>{t("settings.preferences")}</CardTitle>
							<CardDescription>
								{t("settings.preferencesDescription")}
							</CardDescription>
						</CardHeader>
						<CardContent className="space-y-6">
							<div className="flex items-center justify-between">
								<div>
									<div className="font-medium">{t("settings.theme")}</div>
									<div className="text-muted-foreground text-sm">
										Choose your interface theme
									</div>
								</div>
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
										<SelectItem value="system">
											{t("settings.system")}
										</SelectItem>
									</SelectContent>
								</Select>
							</div>

							<div className="flex items-center justify-between">
								<div>
									<div className="font-medium">{t("settings.language")}</div>
									<div className="text-muted-foreground text-sm">
										Select your preferred language
									</div>
								</div>
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
				</div>

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
