import type { models } from "@go/models";
import { Plus, RefreshCw, Settings, X } from "lucide-react";
import {
	useCallback,
	useEffect,
	useRef,
	useState,
	type ReactNode,
} from "react";
import { useTranslation } from "react-i18next";
import { ActionButtons } from "@/components/ActionButtons";
import { EmptyTabState } from "@/components/EmptyTabState";
import { GenerationTabs } from "@/components/GenerationTabs";
import { SessionSelectorModal } from "@/components/SessionSelectorModal";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDocGenerationManager } from "@/hooks/useDocGenerationManager";
import { useDocGenerationStore } from "@/stores/docGeneration";

import type { DocGenerationManager } from "@/hooks/useDocGenerationManager";

type TabContentRendererProps = {
	tabId: string;
	projectId: string;
	project: models.RepoLink;
	mode: "diff" | "single";
	renderGenerationBody: (tabId: string, docManager: DocGenerationManager) => ReactNode;
	canGenerate: boolean;
	canMerge: boolean;
	mergeDisabledReason: string | null;
	onApprove: () => void;
	onGenerate: () => void;
	onReset: () => void;
	onNavigateToGenerations: () => void;
	onNavigateToSettings: () => void;
	onRefreshBranches: () => void;
	onLoadSession: (tabId: string) => void;
	onStartNew: () => void;
};

function TabContentRenderer({
	tabId,
	projectId,
	project,
	mode,
	renderGenerationBody,
	canGenerate,
	canMerge,
	mergeDisabledReason,
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

	// If no session is associated with this tab, show empty state
	if (!docManager.sessionKey) {
		return (
			<div className="flex min-h-0 flex-1 items-center justify-center p-8">
				<EmptyTabState
					onLoadSession={() => onLoadSession(tabId)}
					onStartNew={onStartNew}
				/>
			</div>
		);
	}

	// Tab has a session, render normally
	return (
		<>
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
				<div className="flex flex-wrap items-center gap-2">
					<Button
						onClick={onNavigateToGenerations}
						size="sm"
						type="button"
						variant="outline"
					>
						{t("sidebar.ongoingGenerations")}
					</Button>
					<Button
						onClick={onNavigateToSettings}
						size="sm"
						type="button"
						variant="outline"
					>
						<Settings size={16} />
						{t("common.settings")}
					</Button>
					<Button
						onClick={onRefreshBranches}
						size="sm"
						type="button"
						variant="outline"
					>
						<RefreshCw className="h-4 w-4" />
					</Button>
				</div>
			</header>
			<div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto overflow-x-hidden pr-2">
				{renderGenerationBody(tabId, docManager)}
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
					onApprove={onApprove}
					onCancel={docManager.cancelDocGeneration}
					onGenerate={onGenerate}
					onMerge={docManager.mergeDocs}
					onReset={onReset}
				/>
			)}
		</>
	);
}

export type ProjectDetailTabsSectionProps = {
	projectId: string;
	project: models.RepoLink;
	mode: "diff" | "single";
	renderGenerationBody: (tabId: string, docManager: DocGenerationManager) => ReactNode;
	canGenerate: boolean;
	canMerge: boolean;
	mergeDisabledReason: string | null;
	onApprove: () => void;
	onGenerate: () => void;
	onReset: () => void;
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
	canMerge,
	mergeDisabledReason,
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
	const [sessionSelectorTabId, setSessionSelectorTabId] = useState<string | null>(null);

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

	const handleStartNew = useCallback(() => {
		// User clicks "Start New" - they'll select branches in the normal UI
		// Nothing special to do here, just ensure the tab shows the normal form
	}, []);

	const handleSelectSession = useCallback(
		async (sessionKey: string) => {
			if (!sessionSelectorTabId) {
				return;
			}

			// Parse sessionKey: "projectId:sourceBranch"
			const parts = sessionKey.split(":");
			if (parts.length !== 2) {
				return;
			}

			const [_, sourceBranch] = parts;
			const success = await restoreSession(
				Number(projectId),
				sourceBranch,
				"", // targetBranch not needed for restore
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
								<TabsTrigger
									key={tabId}
									value={tabId}
									className="group flex items-center gap-2 whitespace-nowrap rounded-md px-3 py-1 font-medium text-xs transition data-[state=active]:bg-background data-[state=active]:text-foreground"
								>
									<span>{t("generations.tabLabel", { index: index + 1 })}</span>
									{uiTabs.length > 1 ? (
										<button
											aria-label={t("generations.closeTab", { index: index + 1 })}
											className="rounded p-1 text-muted-foreground transition hover:bg-muted hover:text-foreground"
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
								</TabsTrigger>
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
							key={tabId}
							value={tabId}
							className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
						>
							<TabContentRenderer
								canGenerate={canGenerate}
								canMerge={canMerge}
								mergeDisabledReason={mergeDisabledReason}
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
