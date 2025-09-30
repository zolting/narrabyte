import type { models } from "@go/models";
import {
	CheckLLMInstructions,
	Delete,
	Get,
	ImportLLMInstructions,
	UpdateProjectPaths,
	ValidateDirectory,
} from "@go/services/repoLinkService";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Trash2, TriangleAlert } from "lucide-react";
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

function RepositoryPathField({
	label,
	currentPath,
	newPath,
	onDirectoryChange,
	validationError,
}: {
	label: string;
	currentPath: string;
	newPath: string;
	onDirectoryChange: (path: string) => void;
	validationError: string | null;
}) {
	const { t } = useTranslation();
	const hasChanges = newPath !== currentPath;

	return (
		<div className="space-y-2">
			<div className="block font-medium text-sm">{label}</div>
			<div className="text-muted-foreground text-xs">
				{t("projectSettings.currentPath")}: {currentPath}
			</div>
			<DirectoryPicker onDirectorySelected={onDirectoryChange} />
			{hasChanges && (
				<>
					<div className="rounded bg-background p-2 text-xs">
						<span className="text-muted-foreground">
							{t("projectSettings.newPath")}:{" "}
						</span>
						<span className="font-medium">{newPath}</span>
					</div>
					{validationError && (
						<div className="flex items-center gap-2 rounded bg-destructive/10 p-2 text-destructive text-xs">
							<TriangleAlert size={14} />
							<span>{validationError}</span>
						</div>
					)}
				</>
			)}
		</div>
	);
}

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
	const [docValidationError, setDocValidationError] = useState<string | null>(
		null,
	);
	const [codebaseValidationError, setCodebaseValidationError] = useState<
		string | null
	>(null);

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

	const handleSavePathsSuccess = async () => {
		toast.success(t("projectSettings.pathsUpdated"));
		setDocValidationError(null);
		setCodebaseValidationError(null);
		const updated = (await Get(Number(projectId))) as models.RepoLink;
		setProject(updated);
	};

	const handleSavePathsError = (error: unknown) => {
		console.error("Failed to update paths:", error);
		const errorMessage = error instanceof Error ? error.message : String(error);

		const errorMap: Record<string, () => void> = {
			"missing_git_repo: documentation": () => {
				toast.error(t("projectSettings.noGitRepoDoc"));
				setDocValidationError(t("projectSettings.noGitRepoFound"));
			},
			"missing_git_repo: codebase": () => {
				toast.error(t("projectSettings.noGitRepoCodebase"));
				setCodebaseValidationError(t("projectSettings.noGitRepoFound"));
			},
			"documentation repo path does not exist": () => {
				toast.error(t("projectSettings.docDirNotExist"));
				setDocValidationError(t("projectSettings.dirNotExist"));
			},
			"codebase repo path does not exist": () => {
				toast.error(t("projectSettings.codebaseDirNotExist"));
				setCodebaseValidationError(t("projectSettings.dirNotExist"));
			},
		};

		const matchedError = Object.keys(errorMap).find((key) =>
			errorMessage.includes(key),
		);
		if (matchedError) {
			errorMap[matchedError]();
		} else {
			toast.error(t("projectSettings.pathsUpdateError"));
		}
	};

	const handleSavePaths = async () => {
		if (!project) {
			return;
		}

		setSaving(true);
		try {
			await UpdateProjectPaths(
				Number(project.ID),
				docDirectory,
				codebaseDirectory,
			);
			await handleSavePathsSuccess();
		} catch (error) {
			handleSavePathsError(error);
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

	const getErrorMessageFromCode = (errorCode: string): string => {
		switch (errorCode) {
			case "NO_GIT_REPO":
				return t("projectSettings.noGitRepoFound");
			case "DIR_NOT_EXIST":
				return t("projectSettings.dirNotExist");
			case "EMPTY_PATH":
				return t("projectSettings.dirNotExist");
			default:
				return t("projectSettings.validationFailed");
		}
	};

	const handleDocDirectoryChange = async (path: string) => {
		setDocDirectory(path);
		setDocValidationError(null);

		if (path && path !== project?.DocumentationRepo) {
			try {
				const result = await ValidateDirectory(path);
				if (!result.isValid) {
					setDocValidationError(getErrorMessageFromCode(result.errorCode));
				}
			} catch (error) {
				console.error("Failed to validate documentation directory:", error);
				setDocValidationError(t("projectSettings.validationFailed"));
			}
		}
	};

	const handleCodebaseDirectoryChange = async (path: string) => {
		setCodebaseDirectory(path);
		setCodebaseValidationError(null);

		if (path && path !== project?.CodebaseRepo) {
			try {
				const result = await ValidateDirectory(path);
				if (!result.isValid) {
					setCodebaseValidationError(getErrorMessageFromCode(result.errorCode));
				}
			} catch (error) {
				console.error("Failed to validate codebase directory:", error);
				setCodebaseValidationError(t("projectSettings.validationFailed"));
			}
		}
	};

	const pathsChanged =
		project &&
		(docDirectory !== project.DocumentationRepo ||
			codebaseDirectory !== project.CodebaseRepo);

	const hasValidationErrors =
		docValidationError !== null || codebaseValidationError !== null;

	if (!project) {
		return (
			<div className="p-8 text-muted-foreground">
				Project not found: {projectId}
			</div>
		);
	}

	return (
		<div className="flex flex-col items-center justify-start bg-background p-8 font-mono">
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
							<RepositoryPathField
								currentPath={project.DocumentationRepo}
								label={t("projectSettings.documentationRepo")}
								newPath={docDirectory}
								onDirectoryChange={handleDocDirectoryChange}
								validationError={docValidationError}
							/>

							<RepositoryPathField
								currentPath={project.CodebaseRepo}
								label={t("projectSettings.codebaseRepo")}
								newPath={codebaseDirectory}
								onDirectoryChange={handleCodebaseDirectoryChange}
								validationError={codebaseValidationError}
							/>

							{pathsChanged && (
								<Button
									className="w-full"
									disabled={saving || hasValidationErrors}
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

						<div className="space-y-3 border border-border bg-muted/50 p-4">
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
							<ArrowLeft size={16} />
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
