import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	const { t } = useTranslation();

	return (
		<div className="flex w-full items-center justify-center overflow-hidden bg-background font-mono">
			<div className="flex w-full max-w-5xl p-4">
				<Card className="flex h-[80vh] w-full min-w-0 flex-col overflow-hidden border border-border/60 py-0 shadow-lg">
					<CardHeader className="shrink-0 border-border border-b pt-4 pb-4 text-center">
						<CardTitle className="text-2xl text-foreground">
							{t("common.appName")}
						</CardTitle>
					</CardHeader>
					<CardContent className="flex min-h-0 flex-1 flex-col overflow-hidden px-4 pt-4 pb-4" />
				</Card>
			</div>
		</div>
	);
}
