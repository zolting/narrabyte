import type { models, services } from "@go/models";
import { Plus, RefreshCw, Settings, X } from "lucide-react";
import {
	type ReactNode,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { ActionButtons } from "@/components/ActionButtons";
import { EmptyTabState } from "@/components/EmptyTabState";
import { GenerationTabs } from "@/components/GenerationTabs";
import { SessionSelectorModal } from "@/components/SessionSelectorModal";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "@/components/ui/tooltip";
import type { DocGenerationManager } from "@/hooks/useDocGenerationManager";
import { useDocGenerationManager } from "@/hooks/useDocGenerationManager";
import { useDocGenerationStore } from "@/stores/docGeneration";

function TabLabel({ projectId, tabId }: { projectId: string; tabId: string }) {
	const { t } = useTranslation();
	const branchName = useDocGenerationStore(
		useCallback(
			(state) => {
				const projectKey = String(projectId);
				const projectTabs = state.tabSessions[projectKey] ?? {};
				const hasAnyTabs = Object.keys(projectTabs).length > 0;
				const sessionKey =
					projectTabs[tabId] ??
					(hasAnyTabs ? null : (state.activeSession[projectKey] ?? null));
				if (!sessionKey) {
					return null;
				}
				const source = state.docStates[sessionKey]?.sourceBranch?.trim();
				return source && source.length > 0 ? source : null;
			},
			[projectId, tabId]
		)
	);
	return branchName && branchName.length > 0
		? branchName
		: t("sidebar.newGeneration");
}

type TabContentRendererProps = {
	tabId: string;
	projectId: string;
	project: models.RepoLink;
	mode: "diff" | "single";
	renderGenerationBody: (docManager: DocGenerationManager) => ReactNode;
	canGenerate: boolean;
	currentBranch: string | null;
	hasUncommitted: boolean;
	onApprove: (docManager: DocGenerationManager) => void;
	onGenerate: (tabId: string, docManager: DocGenerationManager) => void;
	onReset: (docManager: DocGenerationManager) => void;
	onNavigateToGenerations: () => void;
	onNavigateToSettings: () => void;
	onRefreshBranches: () => void;
	onLoadSession: (tabId: string) => void;
	onStartNew: (tabId: string) => void;
};

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: none
function TabContentRenderer({
	tabId,
	projectId,
	project,
	mode,
	renderGenerationBody,
	canGenerate,
	currentBranch,
	hasUncommitted,
	onApprove,
	onGenerate,
	onReset,
	onNavigateToGenerations,
	onNavigateToSettings,
	onRefreshBranches,
	onLoadSession,
	onStartNew,
}: TabContentRendererProps) {
	const { t } = useTranslation();
	const docManager = useDocGenerationManager(projectId, tabId);
	const [showSetupWithoutSession, setShowSetupWithoutSession] = useState(false);
	const tabCanGenerate = canGenerate && !docManager.isBusy;

	const { canMerge, mergeDisabledReason } = useMemo(() => {
		if (
			!(
				docManager.docResult &&
				docManager.docsInCodeRepo &&
				docManager.sourceBranch
			) ||
			docManager.isBusy
		) {
			return { canMerge: false, mergeDisabledReason: null as string | null };
		}

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

	useEffect(() => {
		if (docManager.sessionKey) {
			setShowSetupWithoutSession(false);
		}
	}, [docManager.sessionKey]);

	// If no session is associated with this tab, show empty state
	if (!(docManager.sessionKey || showSetupWithoutSession)) {
		return (
			<div className="flex min-h-0 flex-1 items-center justify-center p-8">
				<EmptyTabState
					onLoadSession={() => onLoadSession(tabId)}
					onStartNew={() => {
						setShowSetupWithoutSession(true);
						onStartNew(tabId);
					}}
				/>
			</div>
		);
	}

	let title: string;
	if (docManager.isRunning) {
		title = t("common.generatingDocs");
	} else if (docManager.docResult) {
		title = t("common.docsGenerated");
	} else {
		title = t("common.generateDocs");
	}

	let description: string;
	if (docManager.isRunning) {
		description = t("common.generatingDocsDescription");
	} else if (docManager.docResult) {
		description = t("common.docsGeneratedDescription");
	} else if (mode === "diff") {
		description = t("common.generateDocsDescriptionDiff");
	} else {
		description = t("common.generateDocsDescriptionSingle");
	}

	return (
		<>
			<header className="sticky top-0 z-10 flex shrink-0 items-start justify-between gap-4 bg-card pb-2">
				<div className="space-y-2">
					<h2 className="font-semibold text-foreground text-lg">{title}</h2>
					<p className="text-muted-foreground text-sm">{description}</p>
				</div>
				<div className="flex flex-wrap items-center gap-2">
					{!(docManager.isRunning || docManager.docResult) && (
						<Button
							onClick={onNavigateToGenerations}
							size="sm"
							type="button"
							variant="outline"
						>
							{t("sidebar.ongoingGenerations")}
						</Button>
					)}
					<Button
						onClick={onNavigateToSettings}
						size="sm"
						type="button"
						variant="outline"
					>
						<Settings size={16} />
						{t("common.settings")}
					</Button>
					{!(docManager.isRunning || docManager.docResult) && (
						<TooltipProvider>
							<Tooltip>
								<TooltipTrigger asChild>
									<Button
										onClick={onRefreshBranches}
										size="sm"
										type="button"
										variant="outline"
									>
										<RefreshCw className="h-4 w-4" />
									</Button>
								</TooltipTrigger>
								<TooltipContent>
									<p>{t("common.refreshBranches")}</p>
								</TooltipContent>
							</Tooltip>
						</TooltipProvider>
					)}
				</div>
			</header>
			<div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto overflow-x-hidden pr-2">
				{renderGenerationBody(docManager)}
				{docManager.hasGenerationAttempt && (
					<GenerationTabs
						activeTab={docManager.activeTab}
						docResult={docManager.docResult}
						events={docManager.events}
						projectId={Number(project.ID)}
						sessionKey={docManager.sessionKey}
						setActiveTab={docManager.setActiveTab}
						status={docManager.status}
						todos={docManager.todos}
					/>
				)}
			</div>
			{!docManager.commitCompleted && (
				<ActionButtons
					canGenerate={tabCanGenerate}
					canMerge={canMerge}
					docGenerationError={docManager.docGenerationError}
					docResult={docManager.docResult}
					isBusy={docManager.isBusy}
					isMerging={docManager.isMerging}
					isRunning={docManager.isRunning}
					mergeDisabledReason={mergeDisabledReason}
					onApprove={() => onApprove(docManager)}
					onCancel={docManager.cancelDocGeneration}
					onGenerate={() => onGenerate(tabId, docManager)}
					onMerge={docManager.mergeDocs}
					onReset={() => onReset(docManager)}
				/>
			)}
		</>
	);
}

export type ProjectDetailTabsSectionProps = {
	projectId: string;
	project: models.RepoLink;
	mode: "diff" | "single";
	renderGenerationBody: (docManager: DocGenerationManager) => ReactNode;
	canGenerate: boolean;
	currentBranch: string | null;
	hasUncommitted: boolean;
	onApprove: (docManager: DocGenerationManager) => void;
	onGenerate: (tabId: string, docManager: DocGenerationManager) => void;
	onReset: (docManager: DocGenerationManager) => void;
	onNavigateToGenerations: () => void;
	onNavigateToSettings: () => void;
	onRefreshBranches: () => void;
};

export function ProjectDetailTabsSection({
	projectId,
	project,
	mode,
	renderGenerationBody,
	canGenerate,
	currentBranch,
	hasUncommitted,
	onApprove,
	onGenerate,
	onReset,
	onNavigateToGenerations,
	onNavigateToSettings,
	onRefreshBranches,
}: ProjectDetailTabsSectionProps) {
	const { t } = useTranslation();
	const containerRef = useRef<HTMLDivElement | null>(null);
	const tabCounterRef = useRef(1);
	const [uiTabs, setUiTabs] = useState<string[]>(["tab-1"]);
	const [activeUiTab, setActiveUiTab] = useState("tab-1");
	const [sessionSelectorOpen, setSessionSelectorOpen] = useState(false);
	const [sessionSelectorTabId, setSessionSelectorTabId] = useState<
		string | null
	>(null);

	const restoreSession = useDocGenerationStore((s) => s.restoreSession);

	const addUiTab = useCallback(() => {
		tabCounterRef.current += 1;
		const newTabId = `tab-${tabCounterRef.current}`;
		setUiTabs((prevTabs) => {
			const nextTabs = [...prevTabs, newTabId];
			setActiveUiTab(newTabId);
			return nextTabs;
		});
	}, []);

	const removeUiTab = useCallback(
		(tabId: string) => {
			setUiTabs((prevTabs) => {
				if (prevTabs.length === 1) {
					return prevTabs;
				}
				const filtered = prevTabs.filter((id) => id !== tabId);
				if (filtered.length === prevTabs.length) {
					return prevTabs;
				}
				if (!filtered.includes(activeUiTab)) {
					setActiveUiTab(filtered[0] ?? "tab-1");
				}
				return filtered;
			});
		},
		[activeUiTab]
	);

	useEffect(() => {
		if (!uiTabs.includes(activeUiTab)) {
			setActiveUiTab(uiTabs[0] ?? "tab-1");
		}
	}, [uiTabs, activeUiTab]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: reset tab interface when the project changes.
	useEffect(() => {
		tabCounterRef.current = 1;
		setUiTabs(["tab-1"]);
		setActiveUiTab("tab-1");
	}, [projectId]);

	useEffect(() => {
		if (typeof window === "undefined") {
			return;
		}
		const handler = (event: Event) => {
			const customEvent = event as CustomEvent<{ projectId: string | number }>;
			const targetProjectId = customEvent.detail?.projectId;
			if (targetProjectId === undefined || targetProjectId === null) {
				return;
			}
			if (String(targetProjectId) !== String(projectId)) {
				return;
			}
			addUiTab();
		};
		window.addEventListener("ui:new-generation-tab", handler as EventListener);
		return () => {
			window.removeEventListener(
				"ui:new-generation-tab",
				handler as EventListener
			);
		};
	}, [addUiTab, projectId]);

	const handleLoadSession = useCallback((tabId: string) => {
		setSessionSelectorTabId(tabId);
		setSessionSelectorOpen(true);
	}, []);

	const handleStartNew = useCallback((_tabId: string) => {
		// User clicks "Start New" - they'll select branches in the normal UI
		// Nothing special to do here, just ensure the tab shows the normal form
	}, []);

	const handleSelectSession = useCallback(
		async (session: services.SessionInfo) => {
			if (!sessionSelectorTabId) {
				return;
			}

			const sourceBranch = session.sourceBranch?.trim();
			const targetBranch = session.targetBranch?.trim();

			if (!(sourceBranch && targetBranch)) {
				console.error(
					"Cannot restore session without branch information",
					session
				);
				return;
			}

			const success = await restoreSession(
				Number(projectId),
				sourceBranch,
				targetBranch,
				sessionSelectorTabId
			);

			if (success) {
				setSessionSelectorOpen(false);
				setSessionSelectorTabId(null);
			}
		},
		[projectId, restoreSession, sessionSelectorTabId]
	);

	return (
		<>
			<section
				className="flex min-h-0 flex-1 flex-col gap-6 overflow-hidden rounded-lg border border-border bg-card p-4"
				ref={containerRef}
			>
				<Tabs
					className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
					onValueChange={setActiveUiTab}
					value={activeUiTab}
				>
					<div className="flex items-center justify-between gap-2">
						<TabsList className="flex h-10 items-center gap-1 overflow-x-auto rounded-md bg-muted/60 p-1">
							{uiTabs.map((tabId, index) => (
								<div className="relative" key={tabId}>
									<TabsTrigger
										className="group flex items-center gap-2 whitespace-nowrap rounded-md px-3 py-1 pr-8 font-medium text-xs transition data-[state=active]:bg-background data-[state=active]:text-foreground"
										value={tabId}
									>
										<span className="max-w-[8rem] truncate">
											<TabLabel projectId={projectId} tabId={tabId} />
										</span>
									</TabsTrigger>
									{uiTabs.length > 1 ? (
										<button
											aria-label={t("generations.closeTab", {
												index: index + 1,
											})}
											className="-translate-y-1/2 absolute top-1/2 right-1 rounded p-1 text-muted-foreground transition hover:bg-muted hover:text-foreground"
											onClick={(event) => {
												event.preventDefault();
												event.stopPropagation();
												removeUiTab(tabId);
											}}
											type="button"
										>
											<X className="h-3 w-3" />
											<span className="sr-only">
												{t("generations.closeTab", { index: index + 1 })}
											</span>
										</button>
									) : null}
								</div>
							))}
							<Button
								aria-label={t("generations.addTab")}
								className="h-8 w-8 shrink-0"
								onClick={addUiTab}
								size="icon"
								type="button"
								variant="outline"
							>
								<Plus className="h-4 w-4" />
								<span className="sr-only">{t("generations.addTab")}</span>
							</Button>
						</TabsList>
					</div>
					{uiTabs.map((tabId) => (
						<TabsContent
							className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
							key={tabId}
							value={tabId}
						>
							<TabContentRenderer
								canGenerate={canGenerate}
								currentBranch={currentBranch}
								hasUncommitted={hasUncommitted}
								mode={mode}
								onApprove={onApprove}
								onGenerate={onGenerate}
								onLoadSession={handleLoadSession}
								onNavigateToGenerations={onNavigateToGenerations}
								onNavigateToSettings={onNavigateToSettings}
								onRefreshBranches={onRefreshBranches}
								onReset={onReset}
								onStartNew={handleStartNew}
								project={project}
								projectId={projectId}
								renderGenerationBody={renderGenerationBody}
								tabId={tabId}
							/>
						</TabsContent>
					))}
				</Tabs>
			</section>

			<SessionSelectorModal
				onClose={() => {
					setSessionSelectorOpen(false);
					setSessionSelectorTabId(null);
				}}
				onSelectSession={handleSelectSession}
				open={sessionSelectorOpen}
				projectId={Number(projectId)}
			/>
		</>
	);
}
