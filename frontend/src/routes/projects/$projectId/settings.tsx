import type { models } from "@go/models";
import { ListBranchesByPath } from "@go/services/GitService";
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
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import DirectoryPicker from "@/components/DirectoryPicker";
import { DocumentationBranchSelector } from "@/components/DocumentationBranchSelector";
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
import { pathsShareRoot } from "@/lib/pathUtils";
import { sortBranches } from "@/lib/sortBranches";

export const Route = createFileRoute("/projects/$projectId/settings")({
	component: ProjectSettings,
});

// Component for repository paths section
function RepositoryPathsSection({
	project,
	docDirectory,
	codebaseDirectory,
	docBaseBranch,
	docBranchOptions,
	docValidationError,
	codebaseValidationError,
	docBranchError,
	requiresDocBaseBranch,
	missingDocBaseBranch,
	pathsChanged,
	hasValidationErrors,
	saving,
	handleDocDirectoryChange,
	handleCodebaseDirectoryChange,
	setDocBaseBranch,
	handleSavePaths,
}: {
	project: models.RepoLink;
	docDirectory: string;
	codebaseDirectory: string;
	docBaseBranch: string;
	docBranchOptions: models.BranchInfo[];
	docValidationError: string | null;
	codebaseValidationError: string | null;
	docBranchError: string | null;
	requiresDocBaseBranch: boolean;
	missingDocBaseBranch: boolean;
	pathsChanged: boolean;
	hasValidationErrors: boolean;
	saving: boolean;
	handleDocDirectoryChange: (path: string) => void;
	handleCodebaseDirectoryChange: (path: string) => void;
	setDocBaseBranch: (branch: string) => void;
	handleSavePaths: (
		docDir: string,
		codebaseDir: string,
		baseBranch: string
	) => void;
}) {
	const { t } = useTranslation();

	return (
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

				{requiresDocBaseBranch && (
					<div className="space-y-2">
						<DocumentationBranchSelector
							branches={docBranchOptions}
							description={t(
								"projectSettings.documentationBaseBranchDescription"
							)}
							disabled={!docDirectory || Boolean(docValidationError)}
							onChange={setDocBaseBranch}
							value={docBaseBranch}
						/>
						{docBranchError && (
							<div className="flex items-center gap-2 rounded bg-destructive/10 p-2 text-destructive text-xs">
								<TriangleAlert size={14} />
								<span>{docBranchError}</span>
							</div>
						)}
						{!docBranchError && missingDocBaseBranch && (
							<div className="flex items-center gap-2 rounded bg-destructive/10 p-2 text-destructive text-xs">
								<TriangleAlert size={14} />
								<span>
									{t("projectSettings.documentationBaseBranchRequired")}
								</span>
							</div>
						)}
					</div>
				)}

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
						onClick={() =>
							handleSavePaths(docDirectory, codebaseDirectory, docBaseBranch)
						}
						size="lg"
					>
						{saving
							? t("projectSettings.saving")
							: t("projectSettings.savePaths")}
					</Button>
				)}
			</div>
		</section>
	);
}

// Component for LLM instructions section
function LLMInstructionsSection({
	project,
	hasLLMInstructions,
	llmInstructionsFile,
	setLlmInstructionsFile,
	handleImportLLMInstructions,
}: {
	project: models.RepoLink;
	hasLLMInstructions: boolean;
	llmInstructionsFile: string;
	setLlmInstructionsFile: (file: string) => void;
	handleImportLLMInstructions: () => void;
}) {
	const { t } = useTranslation();

	return (
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
									<span className="font-medium">{llmInstructionsFile}</span>
								</div>
								<Button
									className="w-full"
									onClick={handleImportLLMInstructions}
									size="lg"
								>
									{t("projectSettings.importConfirm")}
								</Button>
							</>
						)}
					</>
				)}
			</div>
		</section>
	);
}

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

// Helper function to get error message from error code
function getErrorMessageFromCode(errorCode: string): {
	key: string;
} {
	switch (errorCode) {
		case "NO_GIT_REPO":
			return { key: "projectSettings.noGitRepoFound" };
		case "DIR_NOT_EXIST":
			return { key: "projectSettings.dirNotExist" };
		case "EMPTY_PATH":
			return { key: "projectSettings.dirNotExist" };
		default:
			return { key: "projectSettings.validationFailed" };
	}
}

// Custom hook for project data management
function useProjectData(projectId: string) {
	const { t } = useTranslation();
	const [project, setProject] = useState<models.RepoLink | null | undefined>(
		undefined
	);
	const [hasLLMInstructions, setHasLLMInstructions] = useState(false);
	const [docDirectory, setDocDirectory] = useState("");
	const [codebaseDirectory, setCodebaseDirectory] = useState("");
	const [docBaseBranch, setDocBaseBranch] = useState("");
	const [docBranchOptions, setDocBranchOptions] = useState<models.BranchInfo[]>(
		[]
	);
	const [docBranchError, setDocBranchError] = useState<string | null>(null);

	const fetchDocBranches = useCallback(
		async (path: string, shouldFetch: boolean) => {
			if (!(path && shouldFetch)) {
				setDocBranchOptions([]);
				setDocBranchError(null);
				return;
			}
			try {
				const branches = await ListBranchesByPath(path);
				setDocBranchOptions(
					sortBranches(branches, { prioritizeMainMaster: true })
				);
				setDocBranchError(null);
			} catch {
				setDocBranchOptions([]);
				setDocBranchError(t("projectSettings.branchLoadFailed"));
			}
		},
		[t]
	);

	useEffect(() => {
		const loadProject = async () => {
			setProject(undefined);
			try {
				const proj = (await Get(Number(projectId))) as models.RepoLink;
				setProject(proj);
				setDocDirectory(proj.DocumentationRepo);
				setCodebaseDirectory(proj.CodebaseRepo);
				setDocBaseBranch(proj.DocumentationBaseBranch ?? "");
				const shared = pathsShareRoot(
					proj.DocumentationRepo,
					proj.CodebaseRepo
				);
				await fetchDocBranches(proj.DocumentationRepo, !shared);

				const hasFile = await CheckLLMInstructions(Number(projectId));
				setHasLLMInstructions(hasFile);
			} catch {
				toast.error(t("projectSettings.loadError"));
				setProject(null);
			}
		};
		loadProject();
	}, [projectId, t, fetchDocBranches]);

	return {
		project,
		setProject,
		hasLLMInstructions,
		setHasLLMInstructions,
		docDirectory,
		setDocDirectory,
		codebaseDirectory,
		setCodebaseDirectory,
		docBaseBranch,
		setDocBaseBranch,
		docBranchOptions,
		setDocBranchOptions,
		docBranchError,
		setDocBranchError,
		fetchDocBranches,
	};
}

// Custom hook for path operations
function usePathOperations(options: {
	projectId: string;
	project: models.RepoLink | null | undefined;
	setProject: (proj: models.RepoLink) => void;
	setDocDirectory: (path: string) => void;
	setCodebaseDirectory: (path: string) => void;
	setDocBaseBranch: (branch: string) => void;
	fetchDocBranches: (path: string, shouldFetch: boolean) => Promise<void>;
}) {
	const { t } = useTranslation();
	const [docValidationError, setDocValidationError] = useState<string | null>(
		null
	);
	const [codebaseValidationError, setCodebaseValidationError] = useState<
		string | null
	>(null);
	const [saving, setSaving] = useState(false);

	const handleSavePathsSuccess = async () => {
		toast.success(t("projectSettings.pathsUpdated"));
		setDocValidationError(null);
		setCodebaseValidationError(null);
		const updated = (await Get(Number(options.projectId))) as models.RepoLink;
		options.setProject(updated);
		options.setDocDirectory(updated.DocumentationRepo);
		options.setCodebaseDirectory(updated.CodebaseRepo);
		options.setDocBaseBranch(updated.DocumentationBaseBranch ?? "");
		const shared = pathsShareRoot(
			updated.DocumentationRepo,
			updated.CodebaseRepo
		);
		await options.fetchDocBranches(updated.DocumentationRepo, !shared);
	};

	const handleSavePathsError = (error: unknown) => {
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
			"documentation base branch is required": () => {
				toast.error(t("projectSettings.documentationBaseBranchRequired"));
			},
		};

		const matchedError = Object.keys(errorMap).find((key) =>
			errorMessage.includes(key)
		);
		if (matchedError) {
			errorMap[matchedError]();
		} else {
			toast.error(t("projectSettings.pathsUpdateError"));
		}
	};

	const handleSavePaths = async (
		docDirectory: string,
		codebaseDirectory: string,
		docBaseBranch: string
	) => {
		if (!options.project) {
			return;
		}

		setSaving(true);
		try {
			await UpdateProjectPaths(
				Number(options.project.ID),
				docDirectory,
				codebaseDirectory,
				docBaseBranch.trim()
			);
			await handleSavePathsSuccess();
		} catch (error) {
			handleSavePathsError(error);
		} finally {
			setSaving(false);
		}
	};

	return {
		docValidationError,
		setDocValidationError,
		codebaseValidationError,
		setCodebaseValidationError,
		saving,
		handleSavePaths,
	};
}

// Custom hook for directory validation
function useDirectoryValidation(options: {
	project: models.RepoLink | null | undefined;
	codebaseDirectory: string;
	docDirectory: string;
	setDocDirectory: (path: string) => void;
	setCodebaseDirectory: (path: string) => void;
	setDocBaseBranch: (branch: string) => void;
	setDocBranchOptions: (opts: models.BranchInfo[]) => void;
	setDocValidationError: (error: string | null) => void;
	setCodebaseValidationError: (error: string | null) => void;
	fetchDocBranches: (path: string, shouldFetch: boolean) => Promise<void>;
}) {
	const { t } = useTranslation();

	const handleDocDirectoryChange = async (path: string) => {
		options.setDocDirectory(path);
		options.setDocValidationError(null);
		options.setDocBaseBranch("");

		if (!path) {
			options.setDocBranchOptions([]);
			return;
		}

		if (path && path !== options.project?.DocumentationRepo) {
			try {
				const result = await ValidateDirectory(path);
				if (!result.isValid) {
					options.setDocBranchOptions([]);
					const errorMsg = getErrorMessageFromCode(result.errorCode);
					options.setDocValidationError(t(errorMsg.key as never));
					return;
				}
			} catch {
				options.setDocValidationError(t("projectSettings.validationFailed"));
				options.setDocBranchOptions([]);
				return;
			}
		}

		const shared = pathsShareRoot(path, options.codebaseDirectory);
		await options.fetchDocBranches(path, !shared);
	};

	const handleCodebaseDirectoryChange = async (path: string) => {
		options.setCodebaseDirectory(path);
		options.setCodebaseValidationError(null);

		if (path && path !== options.project?.CodebaseRepo) {
			try {
				const result = await ValidateDirectory(path);
				if (!result.isValid) {
					const errorMsg = getErrorMessageFromCode(result.errorCode);
					options.setCodebaseValidationError(t(errorMsg.key as never));
				}
			} catch {
				options.setCodebaseValidationError(
					t("projectSettings.validationFailed")
				);
			}
		}

		const shared = pathsShareRoot(options.docDirectory, path);
		await options.fetchDocBranches(options.docDirectory, !shared);
	};

	return {
		handleDocDirectoryChange,
		handleCodebaseDirectoryChange,
	};
}

function ProjectSettings() {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const { projectId } = Route.useParams();

	const {
		project,
		setProject,
		hasLLMInstructions,
		setHasLLMInstructions,
		docDirectory,
		setDocDirectory,
		codebaseDirectory,
		setCodebaseDirectory,
		docBaseBranch,
		setDocBaseBranch,
		docBranchOptions,
		setDocBranchOptions,
		docBranchError,
		setDocBranchError,
		fetchDocBranches,
	} = useProjectData(projectId);

	const {
		docValidationError,
		setDocValidationError,
		codebaseValidationError,
		setCodebaseValidationError,
		saving,
		handleSavePaths,
	} = usePathOperations({
		projectId,
		project,
		setProject,
		setDocDirectory,
		setCodebaseDirectory,
		setDocBaseBranch,
		fetchDocBranches,
	});

	const { handleDocDirectoryChange, handleCodebaseDirectoryChange } =
		useDirectoryValidation({
			project,
			codebaseDirectory,
			docDirectory,
			setDocDirectory,
			setCodebaseDirectory,
			setDocBaseBranch,
			setDocBranchOptions,
			setDocValidationError,
			setCodebaseValidationError,
			fetchDocBranches,
		});

	const [llmInstructionsFile, setLlmInstructionsFile] = useState("");
	const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

	const handleImportLLMInstructions = async () => {
		if (!(project && llmInstructionsFile)) {
			return;
		}

		try {
			await ImportLLMInstructions(project.ID, llmInstructionsFile);
			toast.success(t("projectSettings.llmInstructionsImported"));
			setHasLLMInstructions(true);
			setLlmInstructionsFile("");
		} catch (_error) {
			toast.error(t("projectSettings.llmInstructionsImportError"));
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
		} catch (_error) {
			toast.error(t("sidebar.deleteError"));
		} finally {
			setIsDeleteDialogOpen(false);
		}
	};

	const sharedRepo = useMemo(
		() => pathsShareRoot(docDirectory, codebaseDirectory),
		[docDirectory, codebaseDirectory]
	);

	useEffect(() => {
		if (sharedRepo) {
			setDocBaseBranch("");
			setDocBranchOptions([]);
			setDocBranchError(null);
		}
	}, [sharedRepo, setDocBaseBranch, setDocBranchOptions, setDocBranchError]);

	const requiresDocBaseBranch = Boolean(
		docDirectory && codebaseDirectory && !sharedRepo
	);

	const originalDocBaseBranch = project?.DocumentationBaseBranch ?? "";

	const missingDocBaseBranch =
		requiresDocBaseBranch && docBaseBranch.trim() === "";

	const pathsChanged = Boolean(
		project &&
			(docDirectory !== project.DocumentationRepo ||
				codebaseDirectory !== project.CodebaseRepo ||
				(requiresDocBaseBranch && docBaseBranch !== originalDocBaseBranch))
	);

	const hasValidationErrors =
		docValidationError !== null ||
		codebaseValidationError !== null ||
		missingDocBaseBranch;

	if (project === undefined) {
		return <div className="p-8" />;
	}

	if (!project) {
		return (
			<div className="p-8 text-muted-foreground">
				Project not found: {projectId}
			</div>
		);
	}

	return (
		<div className="flex h-full w-full flex-col overflow-y-auto bg-background p-4 pt-0 font-mono">
			<Card className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border bg-card">
				<CardHeader className="space-y-4">
					<Button
						className="-ml-2 w-fit pl-0 text-muted-foreground hover:bg-transparent hover:text-foreground"
						onClick={() => navigate({ to: `/projects/${projectId}` })}
						size="sm"
						variant="ghost"
					>
						<ArrowLeft className="mr-2 h-4 w-4" />
						{t("common.backToProject")}
					</Button>
					<div className="space-y-1">
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
					</div>
				</CardHeader>
				<CardContent className="space-y-6 overflow-y-auto">
					<RepositoryPathsSection
						codebaseDirectory={codebaseDirectory}
						codebaseValidationError={codebaseValidationError}
						docBaseBranch={docBaseBranch}
						docBranchError={docBranchError}
						docBranchOptions={docBranchOptions}
						docDirectory={docDirectory}
						docValidationError={docValidationError}
						handleCodebaseDirectoryChange={handleCodebaseDirectoryChange}
						handleDocDirectoryChange={handleDocDirectoryChange}
						handleSavePaths={handleSavePaths}
						hasValidationErrors={hasValidationErrors}
						missingDocBaseBranch={missingDocBaseBranch}
						pathsChanged={pathsChanged}
						project={project}
						requiresDocBaseBranch={requiresDocBaseBranch}
						saving={saving}
						setDocBaseBranch={setDocBaseBranch}
					/>

					<LLMInstructionsSection
						handleImportLLMInstructions={handleImportLLMInstructions}
						hasLLMInstructions={hasLLMInstructions}
						llmInstructionsFile={llmInstructionsFile}
						project={project}
						setLlmInstructionsFile={setLlmInstructionsFile}
					/>

					<div className="flex gap-3">
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
						<AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
						<AlertDialogAction
							className="bg-destructive text-destructive-foreground hover:brightness-90 dark:hover:brightness-110"
							onClick={handleDeleteProject}
						>
							{t("common.delete")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
