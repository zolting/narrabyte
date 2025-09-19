import { createFileRoute, Link } from "@tanstack/react-router";
import { GitBranch, Settings } from "lucide-react";
import { useTranslation } from "react-i18next";
import { GitDiffDialog } from "@/components/GitDiffDialog/GitDiffDialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import DemoEvents from "../components/DemoEvents";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	const { t } = useTranslation();

	return (
		<div className="relative flex min-h-screen flex-col items-center justify-center bg-background p-8 font-mono">
			{/* Navigation Buttons */}
			<div className="absolute top-4 right-4 flex gap-2">
				<GitDiffDialog>
					<Button className="text-foreground" size="icon" variant="outline">
						<GitBranch className="h-4 w-4 text-foreground" />
						<span className="sr-only">{t("common.viewDiff")}</span>
					</Button>
				</GitDiffDialog>
				<Button
					asChild
					className="text-foreground"
					size="icon"
					variant="outline"
				>
					<Link to="/settings">
						<Settings className="h-4 w-4 text-foreground" />
						<span className="sr-only">{t("common.settings")}</span>
					</Link>
				</Button>
			</div>

			<Card className="w-full max-w-md">
				<CardHeader className="text-center">
					<CardTitle className="text-xl">Narrabyte</CardTitle>
				</CardHeader>
				<CardContent className="space-y-6">
					<DemoEvents />
				</CardContent>
			</Card>
		</div>
	);
}
