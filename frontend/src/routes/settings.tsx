import { createFileRoute } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const Route = createFileRoute("/settings")({
	component: Settings,
});

function Settings() {
	return (
		<div className="flex min-h-screen flex-col items-center justify-center bg-background p-8 font-mono">
			<Card className="w-full max-w-md">
				<CardHeader className="text-center">
					<CardTitle className="text-2xl">Settings</CardTitle>
				</CardHeader>
				<CardContent className="space-y-6">
					<div className="space-y-4">
						<div className="text-center text-muted-foreground">
							<p>This is the settings page!</p>
							<p className="mt-2 text-sm">TanStack Router is now working</p>
						</div>
						<div className="space-y-2">
							<h3 className="font-semibold text-lg">Available Settings:</h3>
							<ul className="space-y-1 text-muted-foreground text-sm">
								<li>• Theme preferences</li>
								<li>• Directory configurations</li>
								<li>• User preferences</li>
							</ul>
						</div>
					</div>
					<Button
						className="w-full"
						onClick={() => window.history.back()}
						variant="outline"
					>
						Go Back
					</Button>
				</CardContent>
			</Card>
		</div>
	);
}
