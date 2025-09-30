import type { models } from "@go/models";
import { ListApiKeys } from "@go/services/KeyringService";
import { Get } from "@go/services/repoLinkService";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
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
import { useBranchManager } from "@/hooks/useBranchManager";
import { useDocGenerationManager } from "@/hooks/useDocGenerationManager";

export const Route = createFileRoute("/projects/$projectId/")({
	component: ProjectDetailPage,
});

function ProjectDetailPage() {
	const { t } = useTranslation();
	const { projectId } = Route.useParams();
	const [project, setProject] = useState<models.RepoLink | null>(null);
	const [loading, setLoading] = useState(false);
	const [provider, setProvider] = useState<string>("anthropic");
	const [availableProviders, setAvailableProviders] = useState<string[]>([]);
	const containerRef = useRef<HTMLDivElement | null>(null);

	const repoPath = project?.CodebaseRepo;
	const branchManager = useBranchManager(repoPath);
	const docManager = useDocGenerationManager();
	const navigate = useNavigate();

	// Load project data
	useEffect(() => {
		setLoading(true);
		Promise.resolve(Get(Number(projectId)))
			.then((res) => {
				setProject((res as models.RepoLink) ?? null);
			})
			.catch(() => {
				setProject(null);
			})
			.finally(() => {
				setLoading(false);
			});
	}, [projectId]);

	// Load available API keys to determine which providers are available
	useEffect(() => {
		ListApiKeys()
			.then((keys) => {
				const providers = keys.map((k) => k.provider);
				setAvailableProviders(providers);
				// Set default provider to the first available one
				if (providers.length > 0 && !providers.includes(provider)) {
					setProvider(providers[0]);
				}
			})
			.catch((err) => {
				console.error("Failed to load API keys:", err);
				setAvailableProviders([]);
			});
	}, []);

	// Reset everything when component mounts
	useEffect(() => {
		docManager.reset();
		branchManager.resetBranches();
	}, [docManager.reset, branchManager.resetBranches]);

	// Scroll to container when doc generation completes
	useEffect(() => {
		if (docManager.docResult) {
			const node = containerRef.current;
			if (node) {
				node.scrollIntoView({ behavior: "smooth", block: "nearest" });
			}
		}
	}, [docManager.docResult]);

	// Set completed commit info when commit succeeds
	useEffect(() => {
		if (docManager.status === "success" && docManager.commitCompleted) {
			docManager.setCompletedCommit(
				branchManager.sourceBranch || "",
				branchManager.targetBranch || ""
			);
		}
	}, [
		docManager.status,
		docManager.commitCompleted,
		branchManager.sourceBranch,
		branchManager.targetBranch,
		docManager.setCompletedCommit,
	]);

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

	const canCommit = useMemo(() => {
		if (!(project && docManager.docResult)) {
			return false;
		}
		const files = docManager.docResult.files ?? [];
		return files.length > 0 && !docManager.isBusy;
	}, [docManager.docResult, docManager.isBusy, project]);

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
		});
	}, [project, branchManager, docManager, provider]);

	const handleCommit = useCallback(() => {
		if (!(project && docManager.docResult)) {
			return;
		}
		const files = (docManager.docResult.files ?? [])
			.map((file) => file.path)
			.filter((path): path is string =>
				Boolean(path && path.trim().length > 0)
			);
		if (files.length === 0) {
			return;
		}
		docManager.setActiveTab("activity");
		docManager.commitDocGeneration({
			projectId: Number(project.ID),
			branch: docManager.docResult.branch,
			files,
		});
	}, [docManager, project]);

	const handleReset = useCallback(() => {
		docManager.reset();
		branchManager.resetBranches();
	}, [docManager, branchManager]);

	const handleStartNewTask = useCallback(() => {
		handleReset();
	}, [handleReset]);

	const disableControls = docManager.isBusy;

	if (loading) {
		return <div className="p-2 text-muted-foreground text-sm">Loading…</div>;
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
			<div className="flex shrink-0 items-center justify-between">
				<h1 className="flex-1 text-center font-semibold text-foreground text-xl">
					{project.ProjectName}
				</h1>

				<Button
					onClick={() => navigate({ to: `/projects/${projectId}/settings` })}
					size="sm"
					variant="outline"
				>
					<Settings size={16} />
					{t("common.settings")}
				</Button>
			</div>
			<section
				className="flex min-h-0 flex-1 flex-col gap-6 overflow-hidden rounded-lg border border-border bg-card p-4"
				ref={containerRef}
			>
				<header className="shrink-0 space-y-2">
					<h2 className="font-semibold text-foreground text-lg">
						{t("common.generateDocs")}
					</h2>
					<p className="text-muted-foreground text-sm">
						{t("common.generateDocsDescription")}
					</p>
				</header>

				<div className="flex min-h-0 flex-1 flex-col gap-6 overflow-hidden">
					{(() => {
						if (docManager.commitCompleted) {
							return (
								<SuccessPanel
									completedCommitInfo={docManager.completedCommitInfo}
									onStartNewTask={handleStartNewTask}
									sourceBranch={branchManager.sourceBranch}
								/>
							);
						}

						if (docManager.hasGenerationAttempt) {
							return (
								<ComparisonDisplay
									sourceBranch={branchManager.sourceBranch}
									targetBranch={branchManager.targetBranch}
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
										{t("common.provider", "LLM Provider")}
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
									project={project}
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
							</>
						);
					})()}

					{docManager.hasGenerationAttempt && (
						<GenerationTabs
							activeTab={docManager.activeTab}
							docResult={docManager.docResult}
							events={docManager.events}
							setActiveTab={docManager.setActiveTab}
							status={docManager.status}
						/>
					)}
				</div>

				{!docManager.commitCompleted && (
					<ActionButtons
						canCommit={canCommit}
						canGenerate={canGenerate}
						docGenerationError={docManager.docGenerationError}
						docResult={docManager.docResult}
						isBusy={docManager.isBusy}
						isRunning={docManager.isRunning}
						onCancel={docManager.cancelDocGeneration}
						onCommit={handleCommit}
						onGenerate={handleGenerate}
						onReset={handleReset}
					/>
				)}
			</section>
		</div>
	);
}
