import { createFileRoute } from "@tanstack/react-router";
import {useTranslation} from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const Route = createFileRoute("/settings")({
	component: Settings,
});

function Settings() {
	const { t } = useTranslation();

	return (
		<div className="flex min-h-screen flex-col items-center justify-center bg-background p-8 font-mono">
			<Card className="w-full max-w-md">
				<CardHeader className="text-center">
					<CardTitle className="text-2xl">{t("settings.title")}</CardTitle>
				</CardHeader>
				<CardContent className="space-y-6">
					<div className="space-y-4">
						<div className="text-center text-muted-foreground">
							<p>{t("settings.description")}</p>
							<p className="mt-2 text-sm">{t("settings.routerWorking")}</p>
						</div>
						<div className="space-y-2">
							<h3 className="font-semibold text-lg">
								{t("settings.availableSettings")}
							</h3>
							<ul className="space-y-1 text-muted-foreground text-sm">
								<li>• {t("settings.themePreferences")}</li>
								<li>• {t("settings.directoryConfigs")}</li>
								<li>• {t("settings.userPreferences")}</li>
							</ul>
						</div>
					</div>
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
