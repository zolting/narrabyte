import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import AddApiKeyDialog from "@/components/AddApiKeyDialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAppSettingsStore } from "../stores/appSettings";

export const Route = createFileRoute("/settings")({
	component: Settings,
});

function Settings() {
	const { t } = useTranslation();
	const { settings, initialized, loading, setTheme, setLocale } =
		useAppSettingsStore();

	const [dialogOpen, setDialogOpen] = useState(false);
	const theme = settings?.Theme ?? "system";
	const locale = settings?.Locale ?? "en";

	return (
		<div className="flex min-h-screen flex-col items-center justify-center bg-background p-8 font-mono">
			<Card className="w-full max-w-md">
				<CardHeader className="text-center">
					<CardTitle className="text-2xl">{t("settings.title")}</CardTitle>
				</CardHeader>
				<CardContent className="space-y-6">
					{!initialized || loading ? (
						<div className="text-center text-muted-foreground text-sm">
							{t("settings.loading")}
						</div>
					) : (
						<div className="space-y-6">
							<section className="space-y-2">
								<h3 className="font-semibold text-lg">{t("settings.theme")}</h3>
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
							</section>

							<section className="space-y-2">
								<h3 className="font-semibold text-lg">
									{t("settings.language")}
								</h3>
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
							</section>
						</div>
					)}
					<Button onClick={() => setDialogOpen(true)}>Add an API key</Button>
					<AddApiKeyDialog
						open={dialogOpen}
						onClose={() => setDialogOpen(false)}
					/>

					<Button
						className="w-full"
						onClick={() => window.history.back()}
						variant="outline"
					>
						{t("common.goBack")}
					</Button>
				</CardContent>
			</Card>
		</div>
	);
}
