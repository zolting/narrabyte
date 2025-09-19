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
		<div className="flex w-full items-center justify-center bg-background font-mono overflow-hidden">
			<div className="flex w-full max-w-5xl p-4">
				<Card className="flex h-[80vh] w-full min-w-0 flex-col overflow-hidden border border-border/60 shadow-lg py-0">
					<CardHeader className="border-border border-b pb-4 pt-4 text-center shrink-0">
						<CardTitle className="text-2xl text-foreground">
							{t("common.appName")}
						</CardTitle>
					</CardHeader>
					<CardContent className="flex min-h-0 flex-1 flex-col overflow-hidden px-4 pt-4 pb-4">
						<DemoEvents />
					</CardContent>
				</Card>
			</div>
		</div>
	);
}
