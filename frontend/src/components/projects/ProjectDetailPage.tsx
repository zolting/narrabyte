import type { models } from "@go/models";
import {
	GetCurrentBranch,
	HasUncommittedChanges,
} from "@go/services/GitService";
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
import { SuccessPanel } from "@/components/SuccessPanel";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useBranchManager } from "@/hooks/useBranchManager";
import { useDocGenerationManager } from "@/hooks/useDocGenerationManager";

export function ProjectDetailPage({ projectId }: { projectId: string }) {
	const { t } = useTranslation();
	const [project, setProject] = useState<models.RepoLink | null | undefined>(
		undefined
	);
	const [provider, setProvider] = useState<string>("anthropic");
	const [availableProviders, setAvailableProviders] = useState<string[]>([]);
	const [currentBranch, setCurrentBranch] = useState<string | null>(null);
	const [hasUncommitted, setHasUncommitted] = useState<boolean>(false);
	const [userInstructions, setUserInstructions] = useState<string>("");
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
				const providers = keys.map((k) => k.provider);
				setAvailableProviders(providers);
				if (providers.length > 0 && !providers.includes(provider)) {
					setProvider(providers[0]);
				}
			})
			.catch((err) => {
				console.error("Failed to load API keys:", err);
				setAvailableProviders([]);
			});
	}, []);

	useEffect(() => {
		branchManager.resetBranches();
	}, [branchManager.resetBranches]);

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
					branchManager.sourceBranch &&
					branchManager.targetBranch &&
					branchManager.sourceBranch !== branchManager.targetBranch &&
					!docManager.isBusy
			),
		[
			docManager.isBusy,
			project,
			branchManager.sourceBranch,
			branchManager.targetBranch,
		]
	);

	const handleGenerate = useCallback(() => {
		if (
			!(project && branchManager.sourceBranch && branchManager.targetBranch)
		) {
			return;
		}
		branchManager.setSourceOpen(false);
		branchManager.setTargetOpen(false);
		docManager.setActiveTab("activity");
		docManager.startDocGeneration({
			projectId: Number(project.ID),
			sourceBranch: branchManager.sourceBranch,
			targetBranch: branchManager.targetBranch,
			provider,
			userInstructions,
		});
	}, [project, branchManager, docManager, provider, userInstructions]);

	const handleApprove = useCallback(() => {
		docManager.approveCommit();
	}, [docManager]);

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
							{t("common.generateDocsDescription")}
						</p>
					</div>
					<Button
						onClick={() =>
							navigate({
								to: "/projects/$projectId/settings",
								params: { projectId },
							})
						}
						size="sm"
						variant="outline"
					>
						<Settings size={16} />
						{t("common.settings")}
					</Button>
				</header>

				<div className="flex min-h-0 flex-1 flex-col gap-6 overflow-hidden">
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
								<div className="shrink-0 space-y-2">
									<Label
										className="font-medium text-sm"
										htmlFor="provider-select"
									>
										{t("common.llmProvider")}
									</Label>
									<Select
										disabled={
											disableControls || availableProviders.length === 0
										}
										onValueChange={setProvider}
										value={provider}
									>
										<SelectTrigger className="w-full" id="provider-select">
											<SelectValue
												placeholder={t(
													"common.selectProvider",
													"Select a provider"
												)}
											/>
										</SelectTrigger>
										<SelectContent>
											{availableProviders.includes("anthropic") && (
												<SelectItem value="anthropic">
													Anthropic (Claude)
												</SelectItem>
											)}
											{availableProviders.includes("openai") && (
												<SelectItem value="openai">OpenAI</SelectItem>
											)}
											{availableProviders.includes("gemini") && (
												<SelectItem value="gemini">Google Gemini</SelectItem>
											)}
											{availableProviders.includes("openrouter") && (
												<SelectItem value="openrouter">OpenRouter</SelectItem>
											)}
										</SelectContent>
									</Select>
									{availableProviders.length === 0 && (
										<p className="text-muted-foreground text-xs">
											{t(
												"common.noProvidersConfigured",
												"No API keys configured. Please add one in settings."
											)}
										</p>
									)}
								</div>
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
