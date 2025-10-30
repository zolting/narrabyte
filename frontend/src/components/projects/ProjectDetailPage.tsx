import type { models } from "@go/models";
import {
	GetCurrentBranch,
	HasUncommittedChanges,
} from "@go/services/GitService";
import { Delete } from "@go/services/generationSessionService";
import { ListApiKeys } from "@go/services/KeyringService";
import { Get } from "@go/services/repoLinkService";
import { useNavigate } from "@tanstack/react-router";
import { Settings } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ActionButtons } from "@/components/ActionButtons";
import { BranchSelector } from "@/components/BranchSelector";
import { ComparisonDisplay } from "@/components/ComparisonDisplay";
import { GenerationTabs } from "@/components/GenerationTabs";
import { SingleBranchSelector } from "@/components/SingleBranchSelector";
import { SuccessPanel } from "@/components/SuccessPanel";
import { TemplateSelector } from "@/components/TemplateSelector";
import { Button } from "@/components/ui/button";
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
import { Textarea } from "@/components/ui/textarea";
import { useBranchManager } from "@/hooks/useBranchManager";
import { useDocGenerationManager } from "@/hooks/useDocGenerationManager";
import {
	type ModelOption,
	useModelSettingsStore,
} from "@/stores/modelSettings";

export function ProjectDetailPage({ projectId }: { projectId: string }) {
	const { t } = useTranslation();
	const [project, setProject] = useState<models.RepoLink | null | undefined>(
		undefined
	);
	const [modelKey, setModelKey] = useState<string | null>(null);
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
	const [mode, setMode] = useState<"diff" | "single">("diff");
	const [templateInstructions, setTemplateInstructions] = useState<string>("");
	const containerRef = useRef<HTMLDivElement | null>(null);

	const repoPath = project?.CodebaseRepo;
	const branchManager = useBranchManager(repoPath);
	const docManager = useDocGenerationManager(projectId);
	const navigate = useNavigate();

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
	}, []);

	useEffect(() => {
		if (!modelsInitialized) {
			void initModelSettings();
		}
	}, [initModelSettings, modelsInitialized]);

	useEffect(() => {
		branchManager.resetBranches();
	}, [branchManager.resetBranches]);

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
		if (availableModels.length === 0) {
			setModelKey(null);
			return;
		}
		setModelKey((current) => {
			if (current && availableModels.some((model) => model.key === current)) {
				return current;
			}
			return availableModels[0]?.key ?? null;
		});
	}, [availableModels]);

	useEffect(() => {
		if (docManager.docResult) {
			const node = containerRef.current;
			if (node) {
				node.scrollIntoView({ behavior: "smooth", block: "nearest" });
			}
		}
	}, [docManager.docResult]);

	useEffect(() => {
		if (
			docManager.status === "success" &&
			docManager.commitCompleted &&
			branchManager.sourceBranch &&
			branchManager.targetBranch
		) {
			docManager.setCompletedCommit(
				branchManager.sourceBranch,
				branchManager.targetBranch
			);
		}
	}, [
		docManager.status,
		docManager.commitCompleted,
		branchManager.sourceBranch,
		branchManager.targetBranch,
		docManager.setCompletedCommit,
	]);

	// Check current branch and uncommitted changes when docs are in code repo
	useEffect(() => {
		if (repoPath && docManager.docsInCodeRepo) {
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
	}, [repoPath, docManager.docsInCodeRepo]);

	const canGenerate = useMemo(
		() =>
			Boolean(
				project &&
					modelKey &&
					!docManager.isBusy &&
					((mode === "diff" &&
						branchManager.sourceBranch &&
						branchManager.targetBranch &&
						branchManager.sourceBranch !== branchManager.targetBranch) ||
						(mode === "single" &&
							branchManager.sourceBranch &&
							hasInstructionContent))
			),
		[
			branchManager.sourceBranch,
			branchManager.targetBranch,
			docManager.isBusy,
			hasInstructionContent,
			mode,
			modelKey,
			project,
		]
	);

	const handleGenerate = useCallback(() => {
		if (!(project && branchManager.sourceBranch && modelKey)) {
			return;
		}

		const instructions = buildInstructionPayload();

		branchManager.setSourceOpen(false);
		branchManager.setTargetOpen(false);
		docManager.setActiveTab("activity");
		if (mode === "diff") {
			if (!branchManager.targetBranch) {
				return;
			}
			docManager.startDocGeneration({
				projectId: Number(project.ID),
				sourceBranch: branchManager.sourceBranch,
				targetBranch: branchManager.targetBranch,
				modelKey,
				userInstructions: instructions,
			});
		} else if (mode === "single") {
			docManager.startSingleBranchGeneration?.({
				projectId: Number(project.ID),
				sourceBranch: branchManager.sourceBranch,
				targetBranch: "",
				modelKey,
				userInstructions: instructions,
			});
		}
	}, [
		project,
		branchManager,
		docManager,
		modelKey,
		buildInstructionPayload,
		mode,
	]);

	const handleApprove = useCallback(() => {
		docManager.approveCommit();
		const source =
			docManager.sourceBranch ||
			docManager.completedCommitInfo?.sourceBranch ||
			branchManager.sourceBranch ||
			"";
		const target =
			docManager.targetBranch ||
			docManager.completedCommitInfo?.targetBranch ||
			branchManager.targetBranch ||
			"";
		if (source && target) {
			Promise.resolve(Delete(Number(projectId), source, target)).catch(
				() => {}
			);
		}
	}, [
		branchManager.sourceBranch,
		branchManager.targetBranch,
		docManager,
		projectId,
	]);

	const handleReset = useCallback(() => {
		docManager.reset();
		branchManager.resetBranches();
	}, [docManager, branchManager]);

	const handleStartNewTask = useCallback(() => {
		handleReset();
	}, [handleReset]);

	const disableControls = docManager.isBusy;

	// Calculate canMerge and merge disabled reason
	const { canMerge, mergeDisabledReason } = useMemo(() => {
		if (
			!(
				docManager.docResult &&
				docManager.docsInCodeRepo &&
				docManager.sourceBranch
			) ||
			docManager.isBusy
		) {
			return { canMerge: false, mergeDisabledReason: null };
		}

		// Check if currently on source branch with uncommitted changes
		if (currentBranch === docManager.sourceBranch && hasUncommitted) {
			return {
				canMerge: false,
				mergeDisabledReason: "onSourceBranchWithUncommitted",
			};
		}

		return { canMerge: true, mergeDisabledReason: null };
	}, [
		docManager.docResult,
		docManager.docsInCodeRepo,
		docManager.sourceBranch,
		docManager.isBusy,
		currentBranch,
		hasUncommitted,
	]);
	const comparisonSourceBranch =
		docManager.sourceBranch ??
		docManager.completedCommitInfo?.sourceBranch ??
		branchManager.sourceBranch;
	const comparisonTargetBranch =
		docManager.targetBranch ??
		docManager.completedCommitInfo?.targetBranch ??
		branchManager.targetBranch;
	const successSourceBranch =
		docManager.completedCommitInfo?.sourceBranch ??
		docManager.sourceBranch ??
		branchManager.sourceBranch;

	if (project === undefined) {
		return (
			<div className="p-2 text-muted-foreground text-sm">
				{t("common.loading", "Loading project…")}
			</div>
		);
	}

	if (!project) {
		return (
			<div className="p-2 text-muted-foreground text-sm">
				Project not found: {projectId}
			</div>
		);
	}

	return (
		<div className="flex h-[calc(100dvh-4rem)] flex-col gap-6 overflow-hidden p-8">
			<section
				className="flex min-h-0 flex-1 flex-col gap-6 overflow-hidden rounded-lg border border-border bg-card p-4"
				ref={containerRef}
			>
				<header className="sticky top-0 z-10 flex shrink-0 items-start justify-between gap-4 bg-card pb-2">
					<div className="space-y-2">
						<h2 className="font-semibold text-foreground text-lg">
							{t("common.generateDocs")}
						</h2>
						<p className="text-muted-foreground text-sm">
							{mode === "diff"
								? t("common.generateDocsDescriptionDiff")
								: t("common.generateDocsDescriptionSingle")}
						</p>
					</div>
					<div className="flex items-center gap-2">
						<Button
							onClick={() =>
								navigate({
									to: "/projects/$projectId/generations",
									params: { projectId },
								})
							}
							size="sm"
							type="button"
							variant="outline"
						>
							{t("sidebar.ongoingGenerations")}
						</Button>
						<Button
							onClick={() =>
								navigate({
									to: "/projects/$projectId/settings",
									params: { projectId },
								})
							}
							size="sm"
							type="button"
							variant="outline"
						>
							<Settings size={16} />
							{t("common.settings")}
						</Button>
					</div>
				</header>
				<div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto overflow-x-hidden pr-2">
					{(() => {
						if (docManager.commitCompleted) {
							return (
								<SuccessPanel
									completedCommitInfo={docManager.completedCommitInfo}
									onStartNewTask={handleStartNewTask}
									sourceBranch={successSourceBranch}
								/>
							);
						}

						if (docManager.hasGenerationAttempt) {
							return (
								<ComparisonDisplay
									sourceBranch={comparisonSourceBranch}
									targetBranch={comparisonTargetBranch}
								/>
							);
						}

						return (
							<>
								<div className="flex flex-col gap-4 md:flex-row">
									<div className="space-y-2 md:w-1/2">
										<Label
											className="font-medium text-sm"
											htmlFor="model-select"
										>
											{t("common.llmModel")}
										</Label>
										<Select
											disabled={
												disableControls ||
												modelsLoading ||
												availableModels.length === 0
											}
											onValueChange={(value: string) => setModelKey(value)}
											value={modelKey ?? undefined}
										>
											<SelectTrigger className="w-full" id="model-select">
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
									<div className="flex gap-2">
										<Button
											onClick={() => setMode("diff")}
											size="sm"
											type="button"
											variant={mode === "diff" ? "default" : "outline"}
										>
											{t("common.diffMode")}
										</Button>
										<Button
											onClick={() => setMode("single")}
											size="sm"
											type="button"
											variant={mode === "single" ? "default" : "outline"}
										>
											{t("common.singleBranchMode")}
										</Button>
									</div>
								</div>
								{mode === "diff" ? (
									<BranchSelector
										branches={branchManager.branches}
										disableControls={disableControls}
										setSourceBranch={branchManager.setSourceBranch}
										setSourceOpen={branchManager.setSourceOpen}
										setTargetBranch={branchManager.setTargetBranch}
										setTargetOpen={branchManager.setTargetOpen}
										sourceBranch={branchManager.sourceBranch}
										sourceOpen={branchManager.sourceOpen}
										swapBranches={branchManager.swapBranches}
										targetBranch={branchManager.targetBranch}
										targetOpen={branchManager.targetOpen}
									/>
								) : (
									<SingleBranchSelector
										branch={branchManager.sourceBranch}
										branches={branchManager.branches}
										disableControls={disableControls}
										open={branchManager.sourceOpen}
										setBranch={branchManager.setSourceBranch}
										setOpen={branchManager.setSourceOpen}
									/>
								)}
								<div className="space-y-2">
									<Label
										className="font-medium text-sm"
										htmlFor="doc-instructions"
									>
										{t("common.docInstructionsLabel")}
									</Label>
									<Textarea
										className="resize-vertical min-h-[200px] text-xs"
										disabled={disableControls}
										id="doc-instructions"
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
					})()}

					{docManager.hasGenerationAttempt && (
						<GenerationTabs
							activeTab={docManager.activeTab}
							docResult={docManager.docResult}
							events={docManager.events}
							projectId={Number(project.ID)}
							setActiveTab={docManager.setActiveTab}
							status={docManager.status}
						/>
					)}
				</div>
				{!docManager.commitCompleted && (
					<ActionButtons
						canGenerate={canGenerate}
						canMerge={canMerge}
						docGenerationError={docManager.docGenerationError}
						docResult={docManager.docResult}
						isBusy={docManager.isBusy}
						isMerging={docManager.isMerging}
						isRunning={docManager.isRunning}
						mergeDisabledReason={mergeDisabledReason}
						onApprove={handleApprove}
						onCancel={docManager.cancelDocGeneration}
						onGenerate={handleGenerate}
						onMerge={docManager.mergeDocs}
						onReset={handleReset}
					/>
				)}
			</section>
		</div>
	);
}
