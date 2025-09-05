import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
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
										variant={theme === "light" ? "default" : "outline"}
										onClick={() => setTheme("light")}
									>
										{t("settings.light")}
									</Button>
									<Button
										variant={theme === "dark" ? "default" : "outline"}
										onClick={() => setTheme("dark")}
									>
										{t("settings.dark")}
									</Button>
									<Button
										variant={theme === "system" ? "default" : "outline"}
										onClick={() => setTheme("system")}
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
										variant={locale.startsWith("en") ? "default" : "outline"}
										onClick={() => setLocale("en")}
									>
										{t("settings.english")}
									</Button>
									<Button
										variant={locale.startsWith("fr") ? "default" : "outline"}
										onClick={() => setLocale("fr")}
									>
										{t("settings.french")}
									</Button>
								</div>
							</section>
						</div>
					)}

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
