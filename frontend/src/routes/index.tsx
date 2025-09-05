import { createFileRoute, Link } from "@tanstack/react-router";
import { Moon, Settings, Sun } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import logo from "@/assets/images/logo-universal.png";
import DirectoryPicker from "@/components/DirectoryPicker";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Greet, LinkRepositories } from "../../wailsjs/go/main/App";
import DemoEvents from "../components/DemoEvents";
import { useAppSettingsStore } from "../stores/appSettings";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
  const { t, i18n } = useTranslation();
  const [resultText, setResultText] = useState("");
  const [name, setName] = useState("");
  const [docDirectory, setDocDirectory] = useState<string>("");
  const [codebaseDirectory, setCodebaseDirectory] = useState<string>("");
  const { settings, setTheme } = useAppSettingsStore();
  const appTheme = (settings?.Theme ?? "system") as "light" | "dark" | "system";
  const [systemDark, setSystemDark] = useState<boolean>(() =>
    window.matchMedia("(prefers-color-scheme: dark)").matches,
  );

	const updateName = (e: React.ChangeEvent<HTMLInputElement>) =>
		setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);

  const effectiveTheme: "light" | "dark" =
    appTheme === "system" ? (systemDark ? "dark" : "light") : appTheme;

  const toggleTheme = () => {
    setTheme(effectiveTheme === "light" ? "dark" : "light");
  };

	const toggleLanguage = () => {
		const newLang = i18n.language === "en" ? "fr" : "en";
		i18n.changeLanguage(newLang);
	};

  useEffect(() => {
    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => setSystemDark(mql.matches);
    mql.addEventListener("change", onChange);
    return () => {
      mql.removeEventListener("change", onChange);
    };
  }, []);

	const greet = () => {
		Greet(name).then(updateResultText);
	};

	const linkRepositories = async () => {
		if (!(docDirectory && codebaseDirectory)) {
			alert(t("home.selectBothDirectories"));
			return;
		}

		try {
			await LinkRepositories(docDirectory, codebaseDirectory);
			alert(t("home.linkSuccess"));
		} catch (error) {
			console.error("Error linking repositories:", error);
			alert(t("home.linkError"));
		}
	};

	useEffect(() => {
		setResultText(t("home.greeting"));
	}, [t]);

	const isLinkDisabled = !(docDirectory && codebaseDirectory);

	return (
		<div className="relative flex min-h-screen flex-col items-center justify-center bg-background p-8 font-mono">
			{/* Navigation Buttons */}
			<div className="absolute top-4 right-4 flex gap-2">
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
				<Button
					className="text-foreground"
					onClick={toggleLanguage}
					size="icon"
					variant="outline"
				>
					{i18n.language.toUpperCase()}
				</Button>
				<Button
					className="text-foreground"
					onClick={toggleTheme}
					size="icon"
					variant="outline"
				>
          {effectiveTheme === "light" ? (
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
							placeholder={t("home.namePlaceholder")}
							type="text"
							value={name}
						/>
						<Button onClick={greet} size="lg">
							{t("common.greet")}
						</Button>
					</div>

					<div className="space-y-4">
						<div>
							<div className="mb-2 block font-medium text-sm">
								{t("home.docDirectory")}
							</div>
							<DirectoryPicker onDirectorySelected={setDocDirectory} />
						</div>

						<div>
							<div className="mb-2 block font-medium text-sm">
								{t("home.codebaseDirectory")}
							</div>
							<DirectoryPicker onDirectorySelected={setCodebaseDirectory} />
						</div>

						<Button
							className="w-full"
							disabled={isLinkDisabled}
							onClick={linkRepositories}
							size="lg"
						>
							{t("home.linkRepositories")}
						</Button>
					</div>

					<DemoEvents />
				</CardContent>
			</Card>
		</div>
	);
}
