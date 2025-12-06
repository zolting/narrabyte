import { useCallback, useEffect, useMemo, useRef } from "react";
import { useDocGenerationStore } from "@/stores/docGeneration";
import type { TodoItem, ToolEvent } from "@/types/events";

const EMPTY_EVENTS: ToolEvent[] = [];
const EMPTY_TODOS: TodoItem[] = [];

export const useDocGenerationManager = (projectId: string, tabId?: string) => {
	const projectKey = useMemo(() => String(projectId), [projectId]);
	const projectIdNum = useMemo(() => Number(projectId), [projectId]);

	// Session mapping helpers
	const tabSessions = useDocGenerationStore(
		(s) => s.tabSessions[projectKey] ?? null
	);
	const createTabSession = useDocGenerationStore((s) => s.createTabSession);

	// Active session for backward compatibility / default tab
	const activeSession = useDocGenerationStore(
		(s) => s.activeSession[projectKey] ?? null
	);

	const hasAnyTabSessions = useMemo(
		() => (tabSessions ? Object.keys(tabSessions).length > 0 : false),
		[tabSessions]
	);

	// Get the sessionKey for this tab (or fallback to active session for the first tab only)
	const sessionKey = useDocGenerationStore((s) => {
		const tabSessionKey = tabId
			? (s.tabSessions[projectKey]?.[tabId] ?? null)
			: null;
		const active = s.activeSession[projectKey] ?? null;

		// If a tab-specific session exists, always use it
		if (tabSessionKey) {
			return tabSessionKey;
		}

		// If tabs already have assignments, don't fall back to active session for new tabs
		if (tabId && Object.keys(s.tabSessions[projectKey] ?? {}).length > 0) {
			return null;
		}

		// No tab-specific session yet: allow fallback (e.g., landing on project page with an active session)
		if (tabId) {
			return active;
		}

		// No tab context: use the active session for backward compatibility
		return active;
	});

	const persistedFallbackRef = useRef(false);

	// Reset persistence guard when the tab or project changes
	useEffect(() => {
		persistedFallbackRef.current = false;
	}, [projectKey, tabId]);

	// Persist the fallback mapping so subsequent tabs don't inherit the active session
	useEffect(() => {
		if (!(tabId && sessionKey && !hasAnyTabSessions && activeSession)) {
			return;
		}
		if (persistedFallbackRef.current) {
			return;
		}
		persistedFallbackRef.current = true;
		// Map the active session to this tab (typically the initial tab) so new tabs start empty
		createTabSession(projectIdNum, tabId, sessionKey);
	}, [
		activeSession,
		createTabSession,
		hasAnyTabSessions,
		projectIdNum,
		sessionKey,
		tabId,
	]);

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
	const todos = useDocGenerationStore(
		(s) =>
			(sessionKey ? s.docStates[sessionKey]?.todos : EMPTY_TODOS) ?? EMPTY_TODOS
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
	const docsBranch = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.docsBranch : null) ?? null
	);
	const sessionId = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.sessionId : null) ?? null
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

	const reset = useCallback(
		(options?: { deleteDocsBranch?: boolean }) => {
			if (sessionKey) {
				resetDocGeneration(sessionKey, options);
			}
		},
		[sessionKey, resetDocGeneration]
	);

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
			cancelDocGenerationStore(sessionKey);
		}
	}, [cancelDocGenerationStore, sessionKey]);

	const mergeDocs = useCallback(() => {
		if (!(docsInCodeRepo && sessionKey)) {
			return;
		}
		mergeDocsStore({
			sessionKey,
		});
	}, [docsInCodeRepo, mergeDocsStore, sessionKey]);

	return {
		sessionKey,
		sessionId,
		tabId,
		docResult,
		status,
		events,
		todos,
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
		docsBranch,
		mergeDocs,
	};
};
export type DocGenerationManager = ReturnType<typeof useDocGenerationManager>;
