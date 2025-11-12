import { useCallback, useEffect, useMemo, useRef } from "react";
import { useDocGenerationStore } from "@/stores/docGeneration";
import type { DemoEvent } from "@/types/events";

const EMPTY_EVENTS: DemoEvent[] = [];

export const useDocGenerationManager = (projectId: string, tabId?: string) => {
	const projectKey = useMemo(() => String(projectId), [projectId]);
	const projectIdNum = useMemo(() => Number(projectId), [projectId]);

	// Get the sessionKey for this tab (or fallback to active session)
	const sessionKey = useDocGenerationStore((s) => {
		if (tabId) {
			// Only use sessions explicitly associated with this tab
			return s.tabSessions[projectKey]?.[tabId] ?? null;
		}
		// No tab context: use the active session for backward compatibility
		return s.activeSession[projectKey] ?? null;
	});

	// All state is now keyed by sessionKey instead of projectKey
	const docResult = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.result : null) ?? null
	);
	const status = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.status : "idle") ?? "idle"
	);
	const events = useDocGenerationStore(
		(s) =>
			(sessionKey ? s.docStates[sessionKey]?.events : EMPTY_EVENTS) ??
			EMPTY_EVENTS
	);
	const docGenerationError = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.error : null) ?? null
	);
	const activeTab = useDocGenerationStore(
		(s) =>
			(sessionKey ? s.docStates[sessionKey]?.activeTab : "activity") ??
			"activity"
	);
	const commitCompleted = useDocGenerationStore(
		(s) =>
			(sessionKey ? s.docStates[sessionKey]?.commitCompleted : false) ?? false
	);
	const completedCommitInfo = useDocGenerationStore(
		(s) =>
			(sessionKey ? s.docStates[sessionKey]?.completedCommitInfo : null) ?? null
	);
	const sourceBranch = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.sourceBranch : null) ?? null
	);
	const targetBranch = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.targetBranch : null) ?? null
	);
	const docsInCodeRepo = useDocGenerationStore(
		(s) =>
			(sessionKey ? s.docStates[sessionKey]?.docsInCodeRepo : false) ?? false
	);
	const mergeInProgress = useDocGenerationStore(
		(s) =>
			(sessionKey ? s.docStates[sessionKey]?.mergeInProgress : false) ?? false
	);
	const startDocGeneration = useDocGenerationStore((s) => s.start);
	const startSingleBranchGeneration = useDocGenerationStore(
		(s) => s.startFromBranch
	);
	const resetDocGeneration = useDocGenerationStore((s) => s.reset);
	const commitDocGeneration = useDocGenerationStore((s) => s.commit);
	const cancelDocGenerationStore = useDocGenerationStore((s) => s.cancel);
	const setActiveTabStore = useDocGenerationStore((s) => s.setActiveTab);
	const setCompletedCommitInfoStore = useDocGenerationStore(
		(s) => s.setCompletedCommitInfo
	);
	const setCommitCompletedStore = useDocGenerationStore(
		(s) => s.setCommitCompleted
	);
	const mergeDocsStore = useDocGenerationStore((s) => s.mergeDocs);
	const prevDocResultRef = useRef(docResult);
	const prevStatusRef = useRef(status);

	const isRunning = status === "running";
	const isCommitting = status === "committing";
	const isMerging = mergeInProgress;
	const isBusy = isRunning || isCommitting || isMerging;
	const hasGenerationAttempt =
		status !== "idle" || Boolean(docResult) || events.length > 0;

	const setActiveTab = useCallback(
		(tab: "activity" | "review" | "summary") => {
			if (sessionKey) {
				setActiveTabStore(sessionKey, tab);
			}
		},
		[sessionKey, setActiveTabStore]
	);

	// Switch to review tab when LLM completes
	useEffect(() => {
		if (docResult && prevDocResultRef.current !== docResult) {
			setActiveTab("review");
		}
		prevDocResultRef.current = docResult;
	}, [docResult, setActiveTab]);

	// Handle tab switching during status changes
	useEffect(() => {
		if (
			(status === "running" || status === "committing") &&
			prevStatusRef.current !== status
		) {
			setActiveTab("activity");
		}
		prevStatusRef.current = status;
	}, [status, setActiveTab]);

	const reset = useCallback(() => {
		if (sessionKey) {
			resetDocGeneration(projectIdNum, sessionKey);
		}
	}, [projectIdNum, sessionKey, resetDocGeneration]);

	const setCompletedCommit = useCallback(
		(newSourceBranch: string, newTargetBranch: string) => {
			if (sessionKey) {
				setCompletedCommitInfoStore(sessionKey, {
					sourceBranch: newSourceBranch,
					targetBranch: newTargetBranch,
				});
			}
		},
		[sessionKey, setCompletedCommitInfoStore]
	);

	const approveCommit = useCallback(() => {
		if (sessionKey) {
			setCommitCompletedStore(sessionKey, true);
		}
	}, [sessionKey, setCommitCompletedStore]);

	const cancelDocGeneration = useCallback(() => {
		if (sessionKey) {
			cancelDocGenerationStore(projectIdNum, sessionKey);
		}
	}, [cancelDocGenerationStore, projectIdNum, sessionKey]);

	const mergeDocs = useCallback(() => {
		if (!(docsInCodeRepo && docResult?.branch && sessionKey)) {
			return;
		}
		mergeDocsStore({
			projectId: projectIdNum,
			branch: docResult.branch,
			sessionKey,
		});
	}, [
		docsInCodeRepo,
		docResult?.branch,
		mergeDocsStore,
		projectIdNum,
		sessionKey,
	]);

	return {
		sessionKey,
		tabId,
		docResult,
		status,
		events,
		docGenerationError,
		activeTab,
		setActiveTab,
		commitCompleted,
		completedCommitInfo,
		sourceBranch,
		targetBranch,
		isRunning,
		isCommitting,
		isMerging,
		isBusy,
		hasGenerationAttempt,
		startDocGeneration,
		startSingleBranchGeneration,
		commitDocGeneration,
		cancelDocGeneration,
		reset,
		setCompletedCommit,
		approveCommit,
		docsInCodeRepo,
		mergeDocs,
	};
};
export type DocGenerationManager = ReturnType<typeof useDocGenerationManager>;
