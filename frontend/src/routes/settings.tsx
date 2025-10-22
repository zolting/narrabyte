import { createFileRoute } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useEffect, useRef, useState } from "react";
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
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { type AppTheme, useAppSettingsStore } from "@/stores/appSettings";
import { useModelSettingsStore } from "@/stores/modelSettings";

export const Route = createFileRoute("/settings")({
	component: Settings,
});

function Settings() {
	const { t } = useTranslation();
	const { settings, initialized, loading, setTheme, setLocale } =
		useAppSettingsStore();

	const [dialogOpen, setDialogOpen] = useState(false);
	const [editingProvider, setEditingProvider] = useState<string | undefined>(
		undefined,
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

				<Tabs className="space-y-8" defaultValue="general">
					<TabsList className="grid w-full max-w-sm grid-cols-2">
						<TabsTrigger value="general">
							{t("settings.generalTab")}
						</TabsTrigger>
						<TabsTrigger value="models">{t("settings.modelsTab")}</TabsTrigger>
					</TabsList>
					<TabsContent value="general">
						<Card>
							<CardHeader>
								<CardTitle>{t("settings.preferences")}</CardTitle>
								<CardDescription>
									{t("settings.preferencesDescription")}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-6">
								<div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
									<div>
										<div className="font-medium">{t("settings.theme")}</div>
										<div className="text-muted-foreground text-sm">
											{t("settings.selectTheme")}
										</div>
									</div>
									<Select
										disabled={isLoading}
										onValueChange={(value) => setTheme(value as AppTheme)}
										value={theme}
									>
										<SelectTrigger className="w-full sm:w-[180px]">
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="light">
												{t("settings.light")}
											</SelectItem>
											<SelectItem value="dark">{t("settings.dark")}</SelectItem>
											<SelectItem value="system">
												{t("settings.system")}
											</SelectItem>
										</SelectContent>
									</Select>
								</div>

								<div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
									<div>
										<div className="font-medium">{t("settings.language")}</div>
										<div className="text-muted-foreground text-sm">
											{t("settings.selectLanguage")}
										</div>
									</div>
									<Select
										disabled={isLoading}
										onValueChange={setLocale}
										value={locale}
									>
										<SelectTrigger className="w-full sm:w-[180px]">
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
					</TabsContent>
					<TabsContent className="space-y-8" value="models">
						<ApiKeyManager
							onAddClick={handleAddClick}
							onEditClick={handleEditClick}
							ref={apiKeyManagerRef}
						/>
						<ModelsConfiguration />
					</TabsContent>
				</Tabs>

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

function ModelsConfiguration() {
	const { t } = useTranslation();
	const {
		groups,
		initialized,
		loading,
		error,
		init,
		toggleModel,
		toggleProvider,
	} = useModelSettingsStore();

	useEffect(() => {
		if (!initialized) {
			init();
		}
	}, [init, initialized]);

	return (
		<Card>
			<CardHeader>
				<CardTitle>{t("models.title")}</CardTitle>
				<CardDescription>{t("models.description")}</CardDescription>
			</CardHeader>
			<CardContent className="space-y-4">
				{error && (
					<p className="text-destructive text-sm" role="alert">
						{t("models.error")}
					</p>
				)}
				{loading && !initialized && (
					<p className="text-muted-foreground text-sm">{t("models.loading")}</p>
				)}
				{!loading && groups.length === 0 && (
					<p className="text-muted-foreground text-sm">{t("models.empty")}</p>
				)}
				<div className="space-y-3">
					{groups.map((group) => {
						const providerLabel = group.providerName;
						const allEnabled = group.models.every((model) => model.enabled);
						const noneEnabled = group.models.every((model) => !model.enabled);
						const actionLabel = allEnabled
							? t("models.disableAll")
							: t("models.enableAll");
						return (
							<div
								className="space-y-2 rounded-lg border border-border/70 p-3"
								key={group.providerId}
							>
								<div className="flex items-center justify-between gap-2">
									<div className="font-medium text-muted-foreground text-xs uppercase tracking-wide">
										{providerLabel}
									</div>
									<Button
										aria-label={actionLabel}
										disabled={group.models.length === 0}
										onClick={() =>
											toggleProvider(group.providerId, !allEnabled)
										}
										size="sm"
										variant="outline"
									>
										{actionLabel}
									</Button>
								</div>
								{group.models.length > 0 ? (
									<div className="space-y-1.5">
										{group.models.map((model) => (
											<div
												className="flex items-center justify-between gap-3 rounded-md border border-border/50 px-2.5 py-1.5"
												key={model.key}
											>
												<div className="font-medium text-foreground text-sm">
													{model.displayName}
												</div>
												<Switch
													aria-label={t("models.toggleModel", {
														model: model.displayName,
													})}
													checked={model.enabled}
													disabled={!initialized}
													onCheckedChange={(checked) =>
														toggleModel(model.key, checked)
													}
												/>
											</div>
										))}
									</div>
								) : (
									<p className="text-muted-foreground text-sm">
										{t("models.noModelsForProvider")}
									</p>
								)}
								{noneEnabled && (
									<p className="text-amber-600 text-xs">
										{t("models.providerDisabledHint")}
									</p>
								)}
							</div>
						);
					})}
				</div>
			</CardContent>
		</Card>
	);
}
