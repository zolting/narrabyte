import { Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";
import { Greet } from "../wailsjs/go/main/App";
import logo from "./assets/images/logo-universal.png";
import { Button } from "./components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "./components/ui/card";
import { Input } from "./components/ui/input";
import DirectoryPicker from "./DirectoryPicker";

function App() {
	const [resultText, setResultText] = useState(
		"Please enter your name below 👇",
	);
	const [name, setName] = useState("");
	const [selectedDirectory, setSelectedDirectory] = useState<string>("");
	const [theme, setTheme] = useState<"light" | "dark">("light");

	const updateName = (e: React.ChangeEvent<HTMLInputElement>) =>
		setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);
	const handleDirectorySelected = (path: string) => setSelectedDirectory(path);

	// Toggle theme
	const toggleTheme = () => {
		setTheme(theme === "light" ? "dark" : "light");
	};

	// Apply theme to document
	useEffect(() => {
		const root = window.document.documentElement;
		root.classList.remove("light", "dark");
		root.classList.add(theme);
	}, [theme]);

	function greet() {
		Greet(name).then(updateResultText);
	}

	return (
		<div className="relative flex min-h-screen flex-col items-center justify-center bg-background p-8">
			{/* Theme Toggle Button */}
			<Button
				variant="outline"
				size="icon"
				onClick={toggleTheme}
				className="absolute top-4 right-4"
			>
				{theme === "light" ? (
					<Moon className="h-4 w-4" />
				) : (
					<Sun className="h-4 w-4" />
				)}
				<span className="sr-only">Toggle theme</span>
			</Button>

			<Card className="w-full max-w-md">
				<CardHeader className="text-center">
					<img alt="logo" className="mx-auto mb-4 h-20 w-20" src={logo} />
					<CardTitle className="text-xl">{resultText}</CardTitle>
				</CardHeader>
				<CardContent className="space-y-6">
					<div className="flex gap-4">
						<Input
							autoComplete="off"
							name="input"
							onChange={updateName}
							placeholder="Enter your name"
							type="text"
							value={name}
							className="flex-1"
						/>
						<Button onClick={greet} size="lg">
							Greet
						</Button>
					</div>
					<DirectoryPicker onDirectorySelected={handleDirectorySelected} />
					{selectedDirectory && (
						<div className="text-center text-sm text-muted-foreground">
							Selected directory:{" "}
							<code className="bg-muted px-2 py-1 rounded text-xs">
								{selectedDirectory}
							</code>
						</div>
					)}
				</CardContent>
			</Card>
		</div>
	);
}

export default App;
