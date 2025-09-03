import { createFileRoute, Link } from "@tanstack/react-router";
import { Moon, Settings, Sun } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import logo from "@/assets/images/logo-universal.png";
import DirectoryPicker from "@/components/DirectoryPicker";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Greet, LinkRepositories } from "../../wailsjs/go/main/App";
import DemoEvents from "../components/DemoEvents";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	const [resultText, setResultText] = useState(
		"Please enter your name below 👇"
	);
	const [name, setName] = useState("");
	const [docDirectory, setDocDirectory] = useState<string>("");
	const [codebaseDirectory, setCodebaseDirectory] = useState<string>("");
	const [theme, setTheme] = useState<"light" | "dark">("light");

	const updateName = (e: React.ChangeEvent<HTMLInputElement>) =>
		setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);

	const toggleTheme = () => {
		setTheme(theme === "light" ? "dark" : "light");
	};

	useEffect(() => {
		const root = window.document.documentElement;
		root.classList.remove("light", "dark");
		root.classList.add(theme);
	}, [theme]);

	const greet = () => {
		Greet(name).then(updateResultText);
	};

	const linkRepositories = async () => {
		try {
			await LinkRepositories(docDirectory, codebaseDirectory);
			alert("Repositories linked successfully!");
		} catch (error) {
			console.error("Error linking repositories:", error);
			alert("Failed to link repositories");
		}
	};

	const isLinkDisabled = !(docDirectory && codebaseDirectory);

	return (
		<div className="relative flex min-h-screen flex-col items-center justify-center bg-background p-8 font-mono">
			{/* Navigation Buttons */}
			<div className="absolute top-4 right-4 flex gap-2">
				<Button asChild size="icon" variant="outline">
					<Link to="/settings">
						<Settings className="h-4 w-4 text-foreground" />
						<span className="sr-only">Settings</span>
					</Link>
				</Button>
				<Button onClick={toggleTheme} size="icon" variant="outline">
					{theme === "light" ? (
						<Moon className="h-4 w-4 text-foreground" />
					) : (
						<Sun className="h-4 w-4 text-foreground" />
					)}
					<span className="sr-only">Toggle theme</span>
				</Button>
			</div>

			<Card className="w-full max-w-md">
				<CardHeader className="text-center">
					<img alt="logo" className="mx-auto mb-4 h-20 w-20" src={logo} />
					<CardTitle className="text-xl">{resultText}</CardTitle>
				</CardHeader>
				<CardContent className="space-y-6">
					<div className="flex gap-4">
						<Input
							autoComplete="off"
							className="flex-1"
							name="input"
							onChange={updateName}
							placeholder="Enter your name"
							type="text"
							value={name}
						/>
						<Button onClick={greet} size="lg">
							Greet
						</Button>
					</div>

					<div className="space-y-4">
						<div>
							<div className="mb-2 block font-medium text-sm">
								Documentation Directory
							</div>
							<DirectoryPicker onDirectorySelected={setDocDirectory} />
						</div>

						<div>
							<div className="mb-2 block font-medium text-sm">
								Codebase Directory
							</div>
							<DirectoryPicker onDirectorySelected={setCodebaseDirectory} />
						</div>

						<Button
							className="w-full"
							disabled={isLinkDisabled}
							onClick={linkRepositories}
							size="lg"
						>
							Link Repositories
						</Button>
					</div>

					<DemoEvents />
				</CardContent>
			</Card>
		</div>
	);
}
