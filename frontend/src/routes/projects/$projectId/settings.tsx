import type { models } from "@go/models";
import {
	CheckLLMInstructions,
	Delete,
	Get,
	ImportLLMInstructions,
	UpdateProjectPaths,
} from "@go/services/repoLinkService";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import DirectoryPicker from "@/components/DirectoryPicker";
import FilePicker from "@/components/FilePicker";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const Route = createFileRoute("/projects/$projectId/settings")({
	component: ProjectSettings,
});

function ProjectSettings() {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const { projectId } = Route.useParams();
	const [project, setProject] = useState<models.RepoLink | null>(null);
	const [loading, setLoading] = useState(true);
	const [hasLLMInstructions, setHasLLMInstructions] = useState(false);
	const [docDirectory, setDocDirectory] = useState("");
	const [codebaseDirectory, setCodebaseDirectory] = useState("");
	const [llmInstructionsFile, setLlmInstructionsFile] = useState("");
	const [saving, setSaving] = useState(false);
	const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

	useEffect(() => {
		const loadProject = async () => {
			setLoading(true);
			try {
				const proj = (await Get(Number(projectId))) as models.RepoLink;
				setProject(proj);
				setDocDirectory(proj.DocumentationRepo);
				setCodebaseDirectory(proj.CodebaseRepo);

				const hasFile = await CheckLLMInstructions(Number(projectId));
				setHasLLMInstructions(hasFile);
			} catch (error) {
				console.error("Failed to load project:", error);
				toast.error(t("projectSettings.loadError"));
			} finally {
				setLoading(false);
			}
		};
		loadProject();
	}, [projectId, t]);

	const handleSavePaths = async () => {
		if (!project) {
			return;
		}

		setSaving(true);
		try {
			await UpdateProjectPaths(
				Number(project.ID),
				docDirectory,
				codebaseDirectory
			);
			toast.success(t("projectSettings.pathsUpdated"));
			// Reload project
			const updated = (await Get(Number(projectId))) as models.RepoLink;
			setProject(updated);
		} catch (error) {
			console.error("Failed to update paths:", error);
			toast.error(t("projectSettings.pathsUpdateError"));
		} finally {
			setSaving(false);
		}
	};

	const handleImportLLMInstructions = async () => {
		if (!(project && llmInstructionsFile)) {
			return;
		}

		setSaving(true);
		try {
			await ImportLLMInstructions(Number(project.ID), llmInstructionsFile);
			toast.success(t("projectSettings.llmInstructionsImported"));
			setHasLLMInstructions(true);
			setLlmInstructionsFile("");
		} catch (error) {
			console.error("Failed to import LLM instructions:", error);
			toast.error(t("projectSettings.llmInstructionsImportError"));
		} finally {
			setSaving(false);
		}
	};

	const handleDeleteProject = async () => {
		if (!project) {
			return;
		}

		try {
			await Delete(project.ID);
			toast.success(t("sidebar.deleteSuccess"));
			navigate({ to: "/" });
		} catch (error) {
			console.error("Error deleting project:", error);
			toast.error(t("sidebar.deleteError"));
		} finally {
			setIsDeleteDialogOpen(false);
		}
	};

	const pathsChanged =
		project &&
		(docDirectory !== project.DocumentationRepo ||
			codebaseDirectory !== project.CodebaseRepo);

	if (loading) {
		return <div className="p-8 text-muted-foreground">Loading...</div>;
	}

	if (!project) {
		return (
			<div className="p-8 text-muted-foreground">
				Project not found: {projectId}
			</div>
		);
	}

	return (
		<div className="flex min-h-screen flex-col items-center justify-start bg-background p-8 font-mono">
			<Card className="w-full max-w-2xl">
				<CardHeader className="space-y-1">
					<div className="flex items-baseline gap-3">
						<CardTitle className="text-2xl">
							{t("projectSettings.title")}
						</CardTitle>
						<span className="text-base text-muted-foreground">•</span>
						<span className="font-semibold text-foreground text-xl">
							{project.ProjectName}
						</span>
					</div>
					<p className="text-muted-foreground text-sm">
						{t("projectSettings.subtitle")}
					</p>
				</CardHeader>
				<CardContent className="space-y-6">
					<section className="space-y-4">
						<h3 className="font-semibold text-lg">
							{t("projectSettings.repositoryPaths")}
						</h3>

						<div className="space-y-4 rounded-lg border border-border bg-muted/50 p-4">
							<div className="space-y-2">
								<label
									className="block font-medium text-sm"
									htmlFor="doc-directory"
								>
									{t("projectSettings.documentationRepo")}
								</label>
								<div className="text-muted-foreground text-xs">
									{t("projectSettings.currentPath")}:{" "}
									{project.DocumentationRepo}
								</div>
								<DirectoryPicker onDirectorySelected={setDocDirectory} />
								{docDirectory !== project.DocumentationRepo && (
									<div className="rounded bg-background p-2 text-xs">
										<span className="text-muted-foreground">
											{t("projectSettings.newPath")}:{" "}
										</span>
										<span className="font-medium">{docDirectory}</span>
									</div>
								)}
							</div>

							<div className="space-y-2">
								<label
									className="block font-medium text-sm"
									htmlFor="codebase-directory"
								>
									{t("projectSettings.codebaseRepo")}
								</label>
								<div className="text-muted-foreground text-xs">
									{t("projectSettings.currentPath")}: {project.CodebaseRepo}
								</div>
								<DirectoryPicker onDirectorySelected={setCodebaseDirectory} />
								{codebaseDirectory !== project.CodebaseRepo && (
									<div className="rounded bg-background p-2 text-xs">
										<span className="text-muted-foreground">
											{t("projectSettings.newPath")}:{" "}
										</span>
										<span className="font-medium">{codebaseDirectory}</span>
									</div>
								)}
							</div>

							{pathsChanged && (
								<Button
									className="w-full"
									disabled={saving}
									onClick={handleSavePaths}
									size="lg"
								>
									{saving
										? t("projectSettings.saving")
										: t("projectSettings.savePaths")}
								</Button>
							)}
						</div>
					</section>

					<section className="space-y-4">
						<h3 className="font-semibold text-lg">
							{t("projectSettings.llmInstructions")}
						</h3>

						<div className="space-y-2">
							<div className="flex items-center gap-2">
								<span className="text-sm">
									{t("projectSettings.llmInstructionsStatus")}:
								</span>
								<span
									className={`font-semibold text-sm ${hasLLMInstructions ? "text-green-600 dark:text-green-400" : "text-muted-foreground"}`}
								>
									{hasLLMInstructions
										? t("projectSettings.detected")
										: t("projectSettings.notDetected")}
								</span>
							</div>

							{hasLLMInstructions && (
								<div className="text-muted-foreground text-xs">
									{project.DocumentationRepo}/.narrabyte/llm_instructions.*
								</div>
							)}

							{!hasLLMInstructions && (
								<>
									<p className="text-muted-foreground text-sm">
										{t("projectSettings.llmInstructionsDescription")}
									</p>
									<div className="space-y-3 rounded-lg border border-border bg-muted/50 p-4">
										<FilePicker
											accept={{
												label: "LLM Instructions",
												extensions: [
													"md",
													"mdx",
													"txt",
													"json",
													"yaml",
													"yml",
													"prompt",
												],
											}}
											onFileSelected={setLlmInstructionsFile}
										/>
										{llmInstructionsFile && (
											<>
												<div className="rounded bg-background p-2 text-xs">
													<span className="text-muted-foreground">
														{t("projectSettings.selected")}:{" "}
													</span>
													<span className="font-medium">
														{llmInstructionsFile}
													</span>
												</div>
												<Button
													className="w-full"
													disabled={saving}
													onClick={handleImportLLMInstructions}
													size="lg"
												>
													{saving
														? t("projectSettings.importing")
														: t("projectSettings.importConfirm")}
												</Button>
											</>
										)}
									</div>
								</>
							)}
						</div>
					</section>

					<div className="flex gap-3">
						<Button
							className="flex-1"
							onClick={() => navigate({ to: `/projects/${projectId}` })}
							variant="outline"
						>
							{t("common.goBack")}
						</Button>
						<Button
							className="flex-1"
							onClick={() => setIsDeleteDialogOpen(true)}
							variant="destructive"
						>
							<Trash2 size={16} />
							{t("projectSettings.deleteProject")}
						</Button>
					</div>
				</CardContent>
			</Card>

			<AlertDialog
				onOpenChange={setIsDeleteDialogOpen}
				open={isDeleteDialogOpen}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{t("sidebar.deleteProjectTitle")}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{t("sidebar.deleteProjectDescription", {
								projectName: project?.ProjectName,
							})}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>{t("sidebar.cancel")}</AlertDialogCancel>
						<AlertDialogAction
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
							onClick={handleDeleteProject}
						>
							{t("sidebar.delete")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
