import type { models } from "@go/models";
import {
	GetCurrentBranch,
	HasUncommittedChanges,
} from "@go/services/GitService";
import { Delete } from "@go/services/generationSessionService";
import { ListApiKeys } from "@go/services/KeyringService";
import { Get } from "@go/services/repoLinkService";
import { useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useId, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BranchSelector } from "@/components/BranchSelector";
import { ComparisonDisplay } from "@/components/ComparisonDisplay";
import { DocBranchConflictDialog } from "@/components/DocBranchConflictDialog";
import {
	type BranchSelectionState,
	ProjectDetailTabsSection,
} from "@/components/projects/ProjectDetailTabsSection";
import { SingleBranchSelector } from "@/components/SingleBranchSelector";
import { SuccessPanel } from "@/components/SuccessPanel";
import { TemplateSelector } from "@/components/TemplateSelector";

import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectLabel,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useBranchList } from "@/hooks/useBranchManager";
import {
	type DocGenerationManager,
	useDocGenerationManager,
} from "@/hooks/useDocGenerationManager";
import { useAppSettingsStore } from "@/stores/appSettings";
import {
	createSessionKey,
	useDocGenerationStore,
} from "@/stores/docGeneration";
import {
	type ModelOption,
	useModelSettingsStore,
} from "@/stores/modelSettings";

export function ProjectDetailPage({ projectId }: { projectId: string }) {
	const { t } = useTranslation();
	const [project, setProject] = useState<models.RepoLink | null | undefined>(
		undefined
	);
	const [lastSelectedModelKey, setLastSelectedModelKey] = useState<
		string | null
	>(null);
	const [providerKeys, setProviderKeys] = useState<string[]>([]);
	const {
		groups: modelGroups,
		init: initModelSettings,
		initialized: modelsInitialized,
		loading: modelsLoading,
	} = useModelSettingsStore();
	const [currentBranch, setCurrentBranch] = useState<string | null>(null);
	const [hasUncommitted, setHasUncommitted] = useState<boolean>(false);
	const [userInstructions, setUserInstructions] = useState<string>("");
	const [templateInstructions, setTemplateInstructions] = useState<string>("");

	const repoPath = project?.CodebaseRepo;
	const { branches, fetchBranches } = useBranchList(repoPath);
	// Default docManager for auxiliary handlers (uses active session, no specific tab)
	const activeDocManager = useDocGenerationManager(projectId);
	const navigate = useNavigate();
	const docsBranchConflict = useDocGenerationStore((s) => {
		// Get conflict from active session (backward compat)
		const activeSessionKey = s.activeSession[String(projectId)];
		if (activeSessionKey) {
			return s.docStates[activeSessionKey]?.conflict ?? null;
		}
		return null;
	});
	const createTabSession = useDocGenerationStore((s) => s.createTabSession);

	// Read the app's default model preference (if any)
	const { settings: appSettings } = useAppSettingsStore();

	const modelSelectId = useId();
	const docInstructionsId = useId();

	useEffect(() => {
		setProject(undefined);
		Promise.resolve(Get(Number(projectId)))
			.then((res) => {
				setProject((res as models.RepoLink) ?? null);
			})
			.catch(() => {
				setProject(null);
			});
	}, [projectId]);

	useEffect(() => {
		const refreshProviders = () => {
			ListApiKeys()
				.then((keys) => {
					if (!keys) {
						setProviderKeys([]);
						return;
					}
					setProviderKeys(keys.map((k) => k.provider));
				})
				.catch(() => {
					setProviderKeys([]);
				});
			// Réinitialise
			initModelSettings();
		};

		// Load initial
		refreshProviders();

		// Refresh quand les clés ont été changées
		const handler = () => refreshProviders();
		window.addEventListener("narrabyte:keysChanged", handler);
		return () => {
			window.removeEventListener("narrabyte:keysChanged", handler);
		};
	}, [initModelSettings]);

	useEffect(() => {
		if (!modelsInitialized) {
			initModelSettings();
		}
	}, [initModelSettings, modelsInitialized]);

	// Fetch branches on mount
	useEffect(() => {
		fetchBranches();
	}, [fetchBranches]);

	const groupedModelOptions = useMemo(() => {
		if (providerKeys.length === 0) {
			return [] as Array<{
				providerId: string;
				providerName: string;
				models: ModelOption[];
			}>;
		}
		const providers = new Set(providerKeys);
		return modelGroups
			.filter((group) => providers.has(group.providerId))
			.map((group) => ({
				providerId: group.providerId,
				providerName:
					group.providerName?.trim() && group.providerName !== ""
						? group.providerName
						: group.providerId,
				models: group.models.filter((model) => model.enabled),
			}))
			.filter((group) => group.models.length > 0);
	}, [modelGroups, providerKeys]);

	const availableModels = useMemo<ModelOption[]>(
		() => groupedModelOptions.flatMap((group) => group.models),
		[groupedModelOptions]
	);

	const defaultModelKey = useMemo(() => {
		if (availableModels.length === 0) {
			return null;
		}
		const preferred = appSettings?.DefaultModelKey ?? null;
		if (preferred && availableModels.some((m) => m.key === preferred)) {
			return preferred;
		}
		return availableModels[0]?.key ?? null;
	}, [availableModels, appSettings?.DefaultModelKey]);

	const hasInstructionContent = useMemo(() => {
		const template = templateInstructions.trim();
		const user = userInstructions.trim();
		return template.length > 0 || user.length > 0;
	}, [templateInstructions, userInstructions]);

	const buildInstructionPayload = useCallback(() => {
		const sections: string[] = [];
		const template = templateInstructions.trim();
		const user = userInstructions.trim();

		if (template.length > 0) {
			sections.push(
				`<DOCUMENTATION_TEMPLATE>${template}</DOCUMENTATION_TEMPLATE>`
			);
		}
		if (user.length > 0) {
			sections.push(`<USER_INSTRUCTIONS>${user}</USER_INSTRUCTIONS>`);
		}

		return sections.join("");
	}, [templateInstructions, userInstructions]);

	useEffect(() => {
		if (defaultModelKey) {
			setLastSelectedModelKey((current) => current ?? defaultModelKey);
		} else {
			setLastSelectedModelKey(null);
		}
	}, [defaultModelKey]);

	useEffect(() => {
		if (
			activeDocManager.status === "success" &&
			activeDocManager.commitCompleted &&
			activeDocManager.sourceBranch &&
			activeDocManager.targetBranch
		) {
			activeDocManager.setCompletedCommit(
				activeDocManager.sourceBranch,
				activeDocManager.targetBranch
			);
		}
	}, [
		activeDocManager.status,
		activeDocManager.commitCompleted,
		activeDocManager.sourceBranch,
		activeDocManager.targetBranch,
		activeDocManager.setCompletedCommit,
	]);

	// Check current branch and uncommitted changes when docs are in code repo
	useEffect(() => {
		if (repoPath && activeDocManager.docsInCodeRepo) {
			Promise.all([
				GetCurrentBranch(repoPath).catch(() => null),
				HasUncommittedChanges(repoPath).catch(() => false),
			]).then(([branch, uncommitted]) => {
				setCurrentBranch(branch);
				setHasUncommitted(uncommitted);
			});
		} else {
			setCurrentBranch(null);
			setHasUncommitted(false);
		}
	}, [repoPath, activeDocManager.docsInCodeRepo]);

	// Base generation requirements (project + model selected)
	// Branch selection requirements are checked per-tab
	const canGenerateBase = useMemo(
		() => Boolean(project && availableModels.length > 0),
		[availableModels, project]
	);

	const handleGenerate = useCallback(
		(
			tabId: string,
			manager: DocGenerationManager,
			branchSelection: BranchSelectionState,
			mode: "diff" | "single",
			modelKey: string | null
		) => {
			if (!(project && branchSelection.sourceBranch && modelKey)) {
				return;
			}

			const trimmedSourceBranch = branchSelection.sourceBranch.trim();
			if (!trimmedSourceBranch) {
				return;
			}

			const newSessionKey = createSessionKey(
				Number(project.ID),
				trimmedSourceBranch,
				tabId
			);
			createTabSession(Number(project.ID), tabId, newSessionKey);

			const instructions = buildInstructionPayload();

			branchSelection.setSourceOpen(false);
			branchSelection.setTargetOpen(false);
			manager.setActiveTab("activity");
			if (mode === "diff") {
				if (!branchSelection.targetBranch) {
					return;
				}
				manager.startDocGeneration({
					projectId: Number(project.ID),
					projectName: project.ProjectName,
					sourceBranch: trimmedSourceBranch,
					targetBranch: branchSelection.targetBranch,
					modelKey,
					userInstructions: instructions,
					tabId,
				});
			} else if (mode === "single") {
				manager.startSingleBranchGeneration?.({
					projectId: Number(project.ID),
					projectName: project.ProjectName,
					sourceBranch: trimmedSourceBranch,
					targetBranch: "",
					modelKey,
					userInstructions: instructions,
					tabId,
				});
			}
		},
		[project, buildInstructionPayload, createTabSession]
	);

	const handleApprove = useCallback(
		(manager: DocGenerationManager) => {
			manager.approveCommit();
			const source =
				manager.sourceBranch || manager.completedCommitInfo?.sourceBranch || "";
			const target =
				manager.targetBranch || manager.completedCommitInfo?.targetBranch || "";
			if (source && target) {
				Promise.resolve(Delete(Number(projectId), source, target)).catch(() => {
					return;
				});
			}
		},
		[projectId]
	);

	const handleReset = useCallback(
		(manager: DocGenerationManager, branchSelection: BranchSelectionState) => {
			manager.reset();
			branchSelection.resetSelection();
		},
		[]
	);

	const handleStartNewTask = useCallback(
		(manager: DocGenerationManager, branchSelection: BranchSelectionState) => {
			handleReset(manager, branchSelection);
		},
		[handleReset]
	);

	if (project === undefined) {
		return (
			<div className="p-2 text-muted-foreground text-sm">
				{t("common.loading", "Loading project…")}
			</div>
		);
	}

	const renderGenerationSetup = (
		tabDocManager: DocGenerationManager,
		branchSelection: BranchSelectionState,
		mode: "diff" | "single",
		onModeChange: (mode: "diff" | "single") => void,
		modelKey: string | null,
		onModelChange: (modelKey: string | null) => void,
		availableModels: ModelOption[],
		groupedModelOptions: Array<{
			providerId: string;
			providerName: string;
			models: ModelOption[];
		}>,
		modelsLoading: boolean,
		providerKeys: string[]
	) => {
		const disableControls = tabDocManager.isBusy;
		return (
			<>
				<div className="flex flex-col gap-4 md:flex-row">
					<div className="space-y-2 md:w-1/2">
						<Label className="font-medium text-sm" htmlFor={modelSelectId}>
							{t("common.llmModel")}
						</Label>
						<Select
							disabled={
								disableControls || modelsLoading || availableModels.length === 0
							}
							onValueChange={(value: string) => onModelChange(value)}
							value={modelKey ?? undefined}
						>
							<SelectTrigger className="w-full" id={modelSelectId}>
								<SelectValue placeholder={t("common.selectModel")} />
							</SelectTrigger>
							<SelectContent>
								{groupedModelOptions.map((group) => (
									<SelectGroup key={group.providerId}>
										<SelectLabel>{group.providerName}</SelectLabel>
										{group.models.map((model) => (
											<SelectItem key={model.key} value={model.key}>
												{model.displayName}
											</SelectItem>
										))}
									</SelectGroup>
								))}
							</SelectContent>
						</Select>
						{modelsLoading && (
							<p className="text-muted-foreground text-xs">
								{t("models.loading")}
							</p>
						)}
						{!modelsLoading && availableModels.length === 0 && (
							<p className="text-muted-foreground text-xs">
								{providerKeys.length === 0
									? t("common.noProvidersConfigured")
									: t("common.noModelsAvailable")}
							</p>
						)}
					</div>
					<div className="space-y-2 md:w-1/2">
						<TemplateSelector
							setTemplateInstructions={setTemplateInstructions}
						/>
					</div>
				</div>
				<div className="flex items-center gap-2">
					<Label className="text-muted-foreground text-xs">
						{t("common.generationMode")}
					</Label>
					<Tabs
						onValueChange={(v) => onModeChange(v as "diff" | "single")}
						value={mode}
					>
						<TabsList>
							<TabsTrigger value="diff">{t("common.diffMode")}</TabsTrigger>
							<TabsTrigger value="single">
								{t("common.singleBranchMode")}
							</TabsTrigger>
						</TabsList>
					</Tabs>
				</div>
				{mode === "diff" ? (
					<BranchSelector
						branches={branches}
						disableControls={disableControls}
						setSourceBranch={branchSelection.setSourceBranch}
						setSourceOpen={branchSelection.setSourceOpen}
						setTargetBranch={branchSelection.setTargetBranch}
						setTargetOpen={branchSelection.setTargetOpen}
						sourceBranch={branchSelection.sourceBranch}
						sourceOpen={branchSelection.sourceOpen}
						swapBranches={branchSelection.swapBranches}
						targetBranch={branchSelection.targetBranch}
						targetOpen={branchSelection.targetOpen}
					/>
				) : (
					<SingleBranchSelector
						branch={branchSelection.sourceBranch}
						branches={branches}
						disableControls={disableControls}
						open={branchSelection.sourceOpen}
						setBranch={branchSelection.setSourceBranch}
						setOpen={branchSelection.setSourceOpen}
					/>
				)}
				<div className="space-y-2">
					<Label className="font-medium text-sm" htmlFor={docInstructionsId}>
						{t("common.docInstructionsLabel")}
					</Label>
					<Textarea
						className="resize-vertical min-h-[200px] text-xs"
						disabled={disableControls}
						id={docInstructionsId}
						onChange={(e) => setUserInstructions(e.target.value)}
						placeholder={t("common.docInstructionsPlaceholder")}
						value={userInstructions}
					/>
					{mode === "single" && !hasInstructionContent && (
						<p className="text-muted-foreground text-xs">
							{t("common.instructionsRequired")}
						</p>
					)}
				</div>
			</>
		);
	};

	const renderGenerationBody = (
		tabDocManager: DocGenerationManager,
		branchSelection: BranchSelectionState,
		mode: "diff" | "single",
		onModeChange: (mode: "diff" | "single") => void,
		modelKey: string | null,
		onModelChange: (modelKey: string | null) => void,
		availableModels: ModelOption[],
		groupedModelOptions: Array<{
			providerId: string;
			providerName: string;
			models: ModelOption[];
		}>,
		modelsLoading: boolean,
		providerKeys: string[]
	) => {
		const comparisonSourceBranch =
			tabDocManager.sourceBranch ??
			tabDocManager.completedCommitInfo?.sourceBranch ??
			branchSelection.sourceBranch;
		const comparisonTargetBranch =
			tabDocManager.targetBranch ??
			tabDocManager.completedCommitInfo?.targetBranch ??
			branchSelection.targetBranch;
		const successSourceBranch =
			tabDocManager.completedCommitInfo?.sourceBranch ??
			tabDocManager.sourceBranch ??
			branchSelection.sourceBranch;

		if (tabDocManager.commitCompleted) {
			return (
				<SuccessPanel
					completedCommitInfo={tabDocManager.completedCommitInfo}
					onStartNewTask={() =>
						handleStartNewTask(tabDocManager, branchSelection)
					}
					overridenDocsBranch={tabDocManager.docResult?.docsBranch ?? undefined}
					sourceBranch={successSourceBranch}
				/>
			);
		}

		if (tabDocManager.hasGenerationAttempt) {
			return (
				<ComparisonDisplay
					sourceBranch={comparisonSourceBranch}
					targetBranch={comparisonTargetBranch}
				/>
			);
		}

		return renderGenerationSetup(
			tabDocManager,
			branchSelection,
			mode,
			onModeChange,
			modelKey,
			onModelChange,
			availableModels,
			groupedModelOptions,
			modelsLoading,
			providerKeys
		);
	};

	if (!project) {
		return (
			<div className="p-2 text-muted-foreground text-sm">
				{t("common.projectNotFound", { projectId })}
			</div>
		);
	}

	return (
		<div className="flex h-full flex-1 flex-col gap-4 overflow-hidden p-4">
			<ProjectDetailTabsSection
				canGenerateBase={canGenerateBase}
				currentBranch={currentBranch}
				hasInstructionContent={hasInstructionContent}
				hasUncommitted={hasUncommitted}
				onApprove={handleApprove}
				onGenerate={handleGenerate}
				onNavigateToGenerations={() =>
					navigate({
						to: "/projects/$projectId/generations",
						params: { projectId },
					})
				}
				onNavigateToSettings={() =>
					navigate({
						to: "/projects/$projectId/settings",
						params: { projectId },
					})
				}
				onRefreshBranches={fetchBranches}
				onReset={handleReset}
				availableModels={availableModels}
				defaultModelKey={defaultModelKey}
				groupedModelOptions={groupedModelOptions}
				modelsLoading={modelsLoading}
				onModelChange={setLastSelectedModelKey}
				providerKeys={providerKeys}
				project={project}
				projectId={projectId}
				renderGenerationBody={renderGenerationBody}
			/>

			{docsBranchConflict &&
				activeDocManager.sourceBranch &&
				(lastSelectedModelKey ?? defaultModelKey) && (
				<DocBranchConflictDialog
					existingDocsBranch={docsBranchConflict.existingDocsBranch}
					isInProgress={docsBranchConflict.isInProgress}
					mode={docsBranchConflict.mode}
					modelKey={lastSelectedModelKey ?? defaultModelKey ?? undefined}
					open={true}
					projectId={Number(project?.ID ?? projectId)}
					projectName={project.ProjectName}
					proposedDocsBranch={docsBranchConflict.proposedDocsBranch}
					sessionKey={activeDocManager.sessionKey ?? undefined}
					sourceBranch={activeDocManager.sourceBranch}
					targetBranch={activeDocManager.targetBranch ?? undefined}
					userInstructions={buildInstructionPayload()}
				/>
			)}
		</div>
	);
}
