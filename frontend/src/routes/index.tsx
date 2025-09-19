import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import DemoEvents from "../components/DemoEvents";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	const { t } = useTranslation();

	return (
		<div className="flex h-full w-full items-center justify-center bg-background font-mono">
			<Card className="flex h-full w-full min-w-0 max-w-5xl flex-col overflow-hidden border border-border/60 shadow-lg">
				<CardHeader className="border-border border-b pb-6 text-center shrink-0">
					<CardTitle className="text-2xl text-foreground">
						{t("common.appName")}
					</CardTitle>
				</CardHeader>
				<CardContent className="flex min-h-0 flex-1 flex-col overflow-hidden p-6">
					<DemoEvents />
				</CardContent>
			</Card>
		</div>
	);
}
