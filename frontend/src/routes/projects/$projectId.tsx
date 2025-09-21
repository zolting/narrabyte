import type { models } from "@go/models";
import { ListBranchesByPath } from "@go/services/GitService";
import { Get } from "@go/services/repoLinkService";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ArrowRight, CheckIcon, ChevronsUpDownIcon } from "lucide-react";
import { DocGenerationProgressLog } from "@/components/GenerateDocsDialog";
import { DocGenerationResultPanel } from "@/components/DocGenerationResultPanel";
import { Button } from "@/components/ui/button";
import {
	Command,
	CommandEmpty,
	CommandGroup,
	CommandInput,
	CommandItem,
	CommandList,
} from "@/components/ui/command";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { useDocGenerationStore } from "@/stores/docGeneration";

const twTrigger =
	"h-10 w-full bg-card text-card-foreground border border-border " +
	"hover:bg-muted data-[state=open]:bg-muted " +
	"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40";
const twContent =
	"bg-popover text-popover-foreground border border-border shadow-md";

export const Route = createFileRoute("/projects/$projectId")({
	component: ProjectDetailPage,
});

function ProjectDetailPage() {
	const { t } = useTranslation();
	const { projectId } = Route.useParams();
	const [project, setProject] = useState<models.RepoLink | null>(null);
	const [loading, setLoading] = useState(false);
	const docResult = useDocGenerationStore((s) => s.result);

	const status = useDocGenerationStore((s) => s.status);
	const events = useDocGenerationStore((s) => s.events);
	const startDocGeneration = useDocGenerationStore((s) => s.start);
	const resetDocGeneration = useDocGenerationStore((s) => s.reset);
	const docGenerationError = useDocGenerationStore((s) => s.error);
	const isRunning = status === "running";

	const [branches, setBranches] = useState<models.BranchInfo[]>([]);
	const [sourceBranch, setSourceBranch] = useState<string | undefined>();
	const [targetBranch, setTargetBranch] = useState<string | undefined>();
	const [sourceOpen, setSourceOpen] = useState(false);
	const [targetOpen, setTargetOpen] = useState(false);
	const repoPath = project?.CodebaseRepo;
	const [activeTab, setActiveTab] = useState<"activity" | "review">("activity");
	const containerRef = useRef<HTMLDivElement | null>(null);

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

	useEffect(() => {
		resetDocGeneration();
		setSourceBranch(undefined);
		setTargetBranch(undefined);
		setActiveTab("activity");
	}, [projectId, resetDocGeneration]);

	useEffect(() => {
		if (!repoPath) {
			setBranches([]);
			setSourceBranch(undefined);
			setTargetBranch(undefined);
			return;
		}

		let isActive = true;
		ListBranchesByPath(repoPath)
			.then((arr) => {
				if (!isActive) {
					return;
				}
				setBranches(
					[...arr].sort(
						(a, b) =>
							new Date(b.lastCommitDate as unknown as string).getTime() -
							new Date(a.lastCommitDate as unknown as string).getTime(),
					),
				);
			})
			.catch((err) => console.error("failed to fetch branches:", err));

		return () => {
			isActive = false;
		};
	}, [repoPath]);

	const canContinue = useMemo(
		() =>
			Boolean(
				project &&
					sourceBranch &&
					targetBranch &&
					sourceBranch !== targetBranch &&
					!isRunning,
			),
		[isRunning, project, sourceBranch, targetBranch],
	);

	const swapBranches = useCallback(() => {
		setSourceBranch((currentSource) => {
			const next = targetBranch;
			setTargetBranch(currentSource);
			return next;
		});
	}, [targetBranch]);

	const handleGenerate = useCallback(() => {
		if (!project || !sourceBranch || !targetBranch) {
			return;
		}
		setSourceOpen(false);
		setTargetOpen(false);
		setActiveTab("activity");
		void startDocGeneration({
			projectId: Number(project.ID),
			sourceBranch,
			targetBranch,
		});
	}, [project, sourceBranch, startDocGeneration, targetBranch]);

	const handleReset = useCallback(() => {
		resetDocGeneration();
		setSourceBranch(undefined);
		setTargetBranch(undefined);
		setSourceOpen(false);
		setTargetOpen(false);
		setActiveTab("activity");
	}, [resetDocGeneration]);

	const disableControls = isRunning;
	const hasGenerationAttempt = status !== "idle" || Boolean(docResult) || events.length > 0;

	useEffect(() => {
		if (!docResult) {
			return;
		}
		setActiveTab("review");
		const node = containerRef.current;
		if (node) {
			node.scrollIntoView({ behavior: "smooth", block: "start" });
		}
	}, [docResult]);

	useEffect(() => {
		if (status === "running") {
			setActiveTab("activity");
		}
	}, [status]);

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
		<div className="space-y-6">
			<h1 className="text-center font-semibold text-foreground text-xl">
				{project.ProjectName}
			</h1>
			<section
				className="space-y-6 rounded-lg border border-border bg-card p-4"
				ref={containerRef}
			>
				<header className="space-y-2">
					<h2 className="font-semibold text-foreground text-lg">
						{t("common.generateDocs")}
					</h2>
					<p className="text-muted-foreground text-sm">
						{t("common.generateDocsDescription")}
					</p>
				</header>

				<div className="space-y-6">
					<div className="grid gap-2">
						<Label className="mb-1 text-foreground" htmlFor="project-readonly">
							{t("common.project")}
						</Label>
						<div
							className={cn(
								"h-10 w-full rounded-md border border-border bg-card text-card-foreground",
								"flex items-center px-3",
							)}
							id="project-readonly"
						>
							{project.ProjectName}
						</div>
					</div>

					<div className="grid gap-4 sm:grid-cols-2">
						<div className="grid gap-2">
							<div className="flex items-center justify-between">
								<Label
									className="mb-1 text-foreground"
									htmlFor="source-branch-combobox"
								>
									{t("common.sourceBranch")}
								</Label>
								<Button
									className="hover:bg-accent"
									disabled={disableControls || branches.length < 2}
									onClick={swapBranches}
									size="sm"
									type="button"
									variant="secondary"
								>
									{t("common.swapBranches")}
								</Button>
							</div>
							<Popover
								modal={true}
								onOpenChange={setSourceOpen}
								open={sourceOpen}
							>
								<PopoverTrigger asChild>
									<Button
										aria-controls="source-branch-list"
										aria-expanded={sourceOpen}
										className={cn(
											"w-full justify-between hover:text-foreground",
											twTrigger,
										)}
										id="source-branch-combobox"
										role="combobox"
										type="button"
										disabled={disableControls}
										variant="outline"
									>
										{sourceBranch ?? t("common.sourceBranch")}
										<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
									</Button>
								</PopoverTrigger>
								<PopoverContent
									className={cn(
										"w-[var(--radix-popover-trigger-width)] p-0",
										twContent,
									)}
								>
									<Command>
										<CommandInput placeholder="Search branch..." />
										<CommandList
											className="max-h-[200px]"
											id="source-branch-list"
										>
											<CommandEmpty>No branch found.</CommandEmpty>
											<CommandGroup>
												{branches
													.filter((b) => b.name !== targetBranch)
													.map((b) => (
														<CommandItem
															key={b.name}
															onSelect={(currentValue) => {
																setSourceBranch(currentValue);
																setSourceOpen(false);
															}}
															value={b.name}
														>
															<CheckIcon
																className={cn(
																	"mr-2 h-4 w-4",
																	sourceBranch === b.name
																		? "opacity-100"
																		: "opacity-0",
																)}
															/>
															{b.name}
														</CommandItem>
													))}
											</CommandGroup>
										</CommandList>
									</Command>
								</PopoverContent>
							</Popover>
						</div>

						<div className="grid gap-2">
							<Label
								className="mb-1 text-foreground"
								htmlFor="target-branch-combobox"
							>
								{t("common.targetBranch")}
							</Label>
							<Popover
								modal={true}
								onOpenChange={setTargetOpen}
								open={targetOpen}
							>
								<PopoverTrigger asChild>
									<Button
										aria-controls="target-branch-list"
										aria-expanded={targetOpen}
										className={cn(
											"w-full justify-between hover:text-foreground",
											twTrigger,
										)}
										id="target-branch-combobox"
										role="combobox"
										type="button"
										disabled={disableControls}
										variant="outline"
									>
										{targetBranch ?? t("common.targetBranch")}
										<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
									</Button>
								</PopoverTrigger>
								<PopoverContent
									className={cn(
										"w-[var(--radix-popover-trigger-width)] p-0",
										twContent,
									)}
								>
									<Command>
										<CommandInput placeholder="Search branch..." />
										<CommandList
											className="max-h-[200px]"
											id="target-branch-list"
										>
											<CommandEmpty>No branch found.</CommandEmpty>
											<CommandGroup>
												{branches
													.filter((b) => b.name !== sourceBranch)
													.map((b) => (
														<CommandItem
															key={b.name}
															onSelect={(currentValue) => {
																setTargetBranch(currentValue);
																setTargetOpen(false);
															}}
															value={b.name}
														>
															<CheckIcon
																className={cn(
																	"mr-2 h-4 w-4",
																	targetBranch === b.name
																		? "opacity-100"
																		: "opacity-0",
																)}
															/>
															{b.name}
														</CommandItem>
													))}
											</CommandGroup>
										</CommandList>
									</Command>
								</PopoverContent>
							</Popover>
						</div>
					</div>

					{hasGenerationAttempt && (
						<div className="space-y-4">
							<div className="flex flex-wrap gap-2">
								<Button
									aria-pressed={activeTab === "activity"}
									className="sm:w-auto"
									onClick={() => setActiveTab("activity")}
									size="sm"
									type="button"
									variant={activeTab === "activity" ? "default" : "outline"}
								>
									{t("common.recentActivity", "Recent activity")}
								</Button>
								<Button
									aria-pressed={activeTab === "review"}
									className="sm:w-auto"
									onClick={() => setActiveTab("review")}
									size="sm"
									type="button"
									variant={activeTab === "review" ? "default" : "outline"}
								>
									{t("common.review", "Review")}
								</Button>
							</div>
							<div>
								{activeTab === "activity" ? (
									<DocGenerationProgressLog events={events} isRunning={isRunning} />
								) : docResult ? (
									<DocGenerationResultPanel result={docResult} />
								) : (
									<div className="rounded-md border border-dashed border-border p-4 text-sm text-muted-foreground">
										{t("common.noDocumentationChanges", "No documentation changes were produced for this diff.")}
									</div>
								)}
							</div>
						</div>
					)}
				</div>

				<footer className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
					{status === "error" && docGenerationError && (
						<div className="text-destructive text-xs">{docGenerationError}</div>
					)}
					<div className="flex items-center gap-2 sm:justify-end">
						<Button
							className="border-border text-foreground hover:bg-accent"
							onClick={handleReset}
							variant="outline"
						>
							{t("common.reset", "Reset")}
						</Button>
						<Button
							className="gap-2 font-semibold disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
							disabled={!canContinue}
							onClick={handleGenerate}
						>
							{t("common.continue")}
							<ArrowRight className="h-4 w-4" />
						</Button>
					</div>
				</footer>
			</section>
		</div>
	);
}
