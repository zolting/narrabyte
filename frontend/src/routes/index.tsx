import { createFileRoute } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import DemoEvents from "../components/DemoEvents";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	return (
		<div className="flex h-full flex-col items-center justify-center bg-background p-8 font-mono">
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
