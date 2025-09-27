import type { models } from "@go/models";
import { Get } from "@go/services/repoLinkService";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ActionButtons } from "@/components/ActionButtons";
import { BranchSelector } from "@/components/BranchSelector";
import { ComparisonDisplay } from "@/components/ComparisonDisplay";
import { GenerationTabs } from "@/components/GenerationTabs";
import { SuccessPanel } from "@/components/SuccessPanel";
import { useBranchManager } from "@/hooks/useBranchManager";
import { useDocGenerationManager } from "@/hooks/useDocGenerationManager";

export const Route = createFileRoute("/projects/$projectId")({
	component: ProjectDetailPage,
});

function ProjectDetailPage() {
	const { t } = useTranslation();
	const { projectId } = Route.useParams();
	const [project, setProject] = useState<models.RepoLink | null>(null);
	const [loading, setLoading] = useState(false);
	const containerRef = useRef<HTMLDivElement | null>(null);

	const repoPath = project?.CodebaseRepo;
	const branchManager = useBranchManager(repoPath);
	const docManager = useDocGenerationManager();

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
		});
	}, [project, branchManager, docManager]);

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

	const handleRequestDocChanges = useCallback(
		async (message: string) => {
			if (!project) {
				return;
			}
			docManager.setActiveTab("activity");
			await docManager.requestDocChanges({
				projectId: Number(project.ID),
				message,
			});
		},
		[docManager, project]
	);

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
			<h1 className="shrink-0 text-center font-semibold text-foreground text-xl">
				{project.ProjectName}
			</h1>
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
						);
					})()}

					{docManager.hasGenerationAttempt && (
						<GenerationTabs
							activeTab={docManager.activeTab}
							docResult={docManager.docResult}
							events={docManager.events}
							isBusy={docManager.isBusy}
							onRequestChanges={handleRequestDocChanges}
							pendingUserMessage={docManager.pendingUserMessage}
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
