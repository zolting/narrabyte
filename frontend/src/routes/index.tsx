import { createFileRoute, Link } from "@tanstack/react-router";
import { GitBranch, Settings } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import AddProjectDialog from "@/components/AddProjectDialog";
import GenerateDocsDialog from "@/components/GenerateDocsDialog";
import { GitDiffDialog } from "@/components/GitDiffDialog/GitDiffDialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Init } from "../../wailsjs/go/services/GitService";
import { LinkRepositories } from "../../wailsjs/go/services/repoLinkService";
import { Greet } from "../../wailsjs/go/services/userService";
import DemoEvents from "../components/DemoEvents";

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	const { t } = useTranslation();
	const [resultText, setResultText] = useState("");
	const [name, setName] = useState("");
	const [lastProject, setLastProject] = useState<{
		name: string;
		docDirectory: string;
		codebaseDirectory: string;
	} | null>(null);
	const [isAddProjectOpen, setIsAddProjectOpen] = useState(false);
	const [isGenerateDocsOpen, setIsGenerateDocsOpen] = useState(false);

	const updateName = (e: React.ChangeEvent<HTMLInputElement>) =>
		setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);

	const greet = () => {
		Greet(name).then(updateResultText);
	};

	const handleAddProject = async (data: {
		name: string;
		docDirectory: string;
		codebaseDirectory: string;
	}) => {
		if (!(data.docDirectory && data.codebaseDirectory)) {
			alert(t("home.selectBothDirectories"));
			return;
		}
		if (!data.name) {
			alert(t("home.projectNameRequired"));
			return;
		}

		try {
			await LinkRepositories(
				data.name,
				data.docDirectory,
				data.codebaseDirectory,
			);
			alert(t("home.linkSuccess"));
			setIsAddProjectOpen(false);
			setLastProject(data);
		} catch (error) {
			const errorMsg = error instanceof Error ? error.message : String(error);
			if (errorMsg.startsWith("missing_git_repo")) {
				const which = errorMsg.endsWith("documentation")
					? t("projectManager.docDirectory")
					: t("projectManager.codebaseDirectory");
				if (
					window.confirm(
						`${which} n'est pas un dépôt git. Voulez-vous en créer un ?`,
					)
				) {
					// Determine which directory is missing .git
					const dir = errorMsg.endsWith("documentation")
						? data.docDirectory
						: data.codebaseDirectory;

					try {
						await Init(dir);
						// After initializing, try linking again
						await LinkRepositories(
							data.name,
							data.docDirectory,
							data.codebaseDirectory,
						);
						alert(t("home.linkSuccess"));
						setIsAddProjectOpen(false);
						setLastProject(data);
					} catch (initError) {
						console.error("Error initializing git repo:", initError);
						alert("Erreur lors de l'initialisation du dépôt git.");
					}
					return;
				} else {
					return;
				}
			}
			console.error("Error linking repositories:", error);
			alert(t("home.linkError"));
			return;
		}
	};

	useEffect(() => {
		setResultText(t("home.greeting"));
	}, [t]);

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
						<Button
							className="w-full"
							onClick={() => setIsAddProjectOpen(true)}
						>
							{t("home.addProject")}
						</Button>
						<AddProjectDialog
							onClose={() => setIsAddProjectOpen(false)}
							onSubmit={handleAddProject}
							open={isAddProjectOpen}
						/>
						{/*lastProject only used to visualize the changes on screen*/}
						{lastProject && (
							<div className="mt-4 rounded border p-2">
								<div>
									<b>Nom du projet:</b> {lastProject.name}
								</div>
								<div>
									<b>Location du projet:</b> {lastProject.codebaseDirectory}
								</div>
								<div>
									<b>Location de la documentation:</b>{" "}
									{lastProject.docDirectory}
								</div>
							</div>
						)}
					</div>
					<Button onClick={() => setIsGenerateDocsOpen(true)}>
						{t("common.generateDocs")}
					</Button>
					<GenerateDocsDialog
						onClose={() => setIsGenerateDocsOpen(false)}
						open={isGenerateDocsOpen}
					/>
					<DemoEvents />
				</CardContent>
			</Card>
		</div>
	);
}
