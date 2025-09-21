import type { models } from "@go/models";
import { ListBranchesByPath } from "@go/services/GitService";
import { Get } from "@go/services/repoLinkService";
import { createFileRoute } from "@tanstack/react-router";
import {
	ArrowRight,
	ArrowRightLeft,
	CheckIcon,
	ChevronsUpDownIcon,
} from "lucide-react";
import {
	useCallback,
	useEffect,
	useId,
	useMemo,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { DocGenerationResultPanel } from "@/components/DocGenerationResultPanel";
import { DocGenerationProgressLog } from "@/components/GenerateDocsDialog";
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
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
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
	const commitDocGeneration = useDocGenerationStore((s) => s.commit);
	const docGenerationError = useDocGenerationStore((s) => s.error);
	const isRunning = status === "running";
	const isCommitting = status === "committing";
	const isBusy = isRunning || isCommitting;

	const [branches, setBranches] = useState<models.BranchInfo[]>([]);
	const [sourceBranch, setSourceBranch] = useState<string | undefined>();
	const [targetBranch, setTargetBranch] = useState<string | undefined>();
	const [sourceOpen, setSourceOpen] = useState(false);
	const [targetOpen, setTargetOpen] = useState(false);
	const repoPath = project?.CodebaseRepo;
	const [activeTab, setActiveTab] = useState<"activity" | "review">("activity");
	const containerRef = useRef<HTMLDivElement | null>(null);

	const projectInputId = useId();
	const sourceBranchComboboxId = useId();
	const sourceBranchListId = useId();
	const targetBranchComboboxId = useId();
	const targetBranchListId = useId();

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
	}, [resetDocGeneration]);

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
							new Date(a.lastCommitDate as unknown as string).getTime()
					)
				);
			})
			.catch((err) => console.error("failed to fetch branches:", err));

		return () => {
			isActive = false;
		};
	}, [repoPath]);

	const canGenerate = useMemo(
		() =>
			Boolean(
				project &&
					sourceBranch &&
					targetBranch &&
					sourceBranch !== targetBranch &&
					!isBusy
			),
		[isBusy, project, sourceBranch, targetBranch]
	);

	const canCommit = useMemo(() => {
		if (!project || !docResult) {
			return false;
		}
		const files = docResult.files ?? [];
		return files.length > 0 && !isBusy;
	}, [docResult, isBusy, project]);

	const swapBranches = useCallback(() => {
		setSourceBranch((currentSource) => {
			const next = targetBranch;
			setTargetBranch(currentSource);
			return next;
		});
	}, [targetBranch]);

	const handleGenerate = useCallback(() => {
		if (!(project && sourceBranch && targetBranch)) {
			return;
		}
		setSourceOpen(false);
		setTargetOpen(false);
		setActiveTab("activity");
		startDocGeneration({
			projectId: Number(project.ID),
			sourceBranch,
			targetBranch,
		});
	}, [project, sourceBranch, startDocGeneration, targetBranch]);

	const handleCommit = useCallback(() => {
		if (!(project && docResult)) {
			return;
		}
		const files = (docResult.files ?? [])
			.map((file) => file.path)
			.filter((path): path is string => Boolean(path && path.trim().length > 0));
		if (files.length === 0) {
			return;
		}
		setActiveTab("activity");
		commitDocGeneration({
			projectId: Number(project.ID),
			branch: docResult.branch,
			files,
		});
	}, [commitDocGeneration, docResult, project]);

	const handleReset = useCallback(() => {
		resetDocGeneration();
		setSourceBranch(undefined);
		setTargetBranch(undefined);
		setSourceOpen(false);
		setTargetOpen(false);
		setActiveTab("activity");
	}, [resetDocGeneration]);

	const disableControls = isBusy;
	const hasGenerationAttempt =
		status !== "idle" || Boolean(docResult) || events.length > 0;

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
		if (status === "running" || status === "committing") {
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
		<div className="flex h-[calc(100dvh-4rem)] flex-col gap-6 overflow-hidden p-8">
			<h1 className="shrink-0 text-center font-semibold text-foreground text-xl">
				{project.ProjectName}
			</h1>
			<section
				className="flex flex-1 min-h-0 flex-col gap-6 overflow-hidden rounded-lg border border-border bg-card p-4"
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

				<div className="flex flex-1 min-h-0 flex-col gap-6 overflow-hidden">
					<div className="grid shrink-0 gap-2">
						<Label className="mb-1 text-foreground" htmlFor={projectInputId}>
							{t("common.project")}
						</Label>
						<div
							className={cn(
								"h-10 w-full rounded-md border border-border bg-card text-card-foreground",
								"flex items-center px-3"
							)}
							id={projectInputId}
						>
							{project.ProjectName}
						</div>
					</div>

					<div className="grid shrink-0 grid-cols-[1fr_auto_1fr] items-end gap-4">
						<div className="grid gap-2">
							<Label
								className="mb-1 text-foreground"
								htmlFor={sourceBranchComboboxId}
							>
								{t("common.sourceBranch")}
							</Label>
							<Popover
								modal={true}
								onOpenChange={setSourceOpen}
								open={sourceOpen}
							>
								<PopoverTrigger asChild>
									<Button
										aria-controls={sourceBranchListId}
										aria-expanded={sourceOpen}
										className={cn(
											"w-full justify-between hover:text-foreground",
											twTrigger
										)}
										disabled={disableControls}
										id={sourceBranchComboboxId}
										role="combobox"
										type="button"
										variant="outline"
									>
										{sourceBranch ?? t("common.sourceBranch")}
										<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
									</Button>
								</PopoverTrigger>
								<PopoverContent
									className={cn(
										"w-[var(--radix-popover-trigger-width)] p-0",
										twContent
									)}
								>
									<Command>
										<CommandInput placeholder="Search branch..." />
										<CommandList
											className="max-h-[200px]"
											id={sourceBranchListId}
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
																		: "opacity-0"
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

						<Button
							aria-label={t("common.swapBranches")}
							className="h-10 w-10 p-1 hover:bg-accent"
							disabled={disableControls || branches.length < 2}
							onClick={swapBranches}
							type="button"
							variant="secondary"
						>
							<ArrowRightLeft className="h-4 w-4" />
						</Button>

						<div className="grid gap-2">
							<Label
								className="mb-1 text-foreground"
								htmlFor={targetBranchComboboxId}
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
										aria-controls={targetBranchListId}
										aria-expanded={targetOpen}
										className={cn(
											"w-full justify-between hover:text-foreground",
											twTrigger
										)}
										disabled={disableControls}
										id={targetBranchComboboxId}
										role="combobox"
										type="button"
										variant="outline"
									>
										{targetBranch ?? t("common.targetBranch")}
										<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
									</Button>
								</PopoverTrigger>
								<PopoverContent
									className={cn(
										"w-[var(--radix-popover-trigger-width)] p-0",
										twContent
									)}
								>
									<Command>
										<CommandInput placeholder="Search branch..." />
										<CommandList
											className="max-h-[200px]"
											id={targetBranchListId}
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
																		: "opacity-0"
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
						<div className="flex flex-1 min-h-0 flex-col gap-4 overflow-hidden">
							<div className="flex shrink-0 flex-wrap gap-2">
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
							<div className="flex flex-1 min-h-0 flex-col overflow-hidden">
								{(() => {
									if (activeTab === "activity") {
										return (
											<DocGenerationProgressLog
												events={events}
												status={status}
											/>
										);
									}

									if (docResult) {
										return <DocGenerationResultPanel result={docResult} />;
									}

									return (
										<div className="rounded-md border border-border border-dashed p-4 text-muted-foreground text-sm">
											{t(
												"common.noDocumentationChanges",
												"No documentation changes were produced for this diff."
											)}
										</div>
									);
								})()}
							</div>
						</div>
					)}
				</div>

				<footer className="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
					{status === "error" && docGenerationError && (
						<div className="text-destructive text-xs">{docGenerationError}</div>
					)}
					<div className="flex items-center gap-2 sm:justify-end">
						<Button
							className="border-border text-foreground hover:bg-accent"
							disabled={isBusy}
							onClick={handleReset}
							variant="outline"
						>
							{t("common.reset", "Reset")}
						</Button>
						{docResult ? (
							<Button
								className="gap-2 font-semibold disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
								disabled={!canCommit}
								onClick={handleCommit}
							>
								{t("common.commit", "Commit")}
								<ArrowRight className="h-4 w-4" />
							</Button>
						) : (
							<Button
								className="gap-2 font-semibold disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
								disabled={!canGenerate}
								onClick={handleGenerate}
							>
								{t("common.generateDocs", "Generate docs")}
								<ArrowRight className="h-4 w-4" />
							</Button>
						)}
					</div>
				</footer>
			</section>
		</div>
	);
}
