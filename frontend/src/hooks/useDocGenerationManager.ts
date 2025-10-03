import { useCallback, useEffect, useMemo, useRef } from "react";
import { useDocGenerationStore } from "@/stores/docGeneration";
import type { DemoEvent } from "@/types/events";

const EMPTY_EVENTS: DemoEvent[] = [];

export const useDocGenerationManager = (projectId: string) => {
	const projectKey = useMemo(() => String(projectId), [projectId]);
	const docResult = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.result ?? null
	);
	const status = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.status ?? "idle"
	);
	const events = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.events ?? EMPTY_EVENTS
	);
	const docGenerationError = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.error ?? null
	);
	const activeTab = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.activeTab ?? "activity"
	);
	const commitCompleted = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.commitCompleted ?? false
	);
	const completedCommitInfo = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.completedCommitInfo ?? null
	);
	const sourceBranch = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.sourceBranch ?? null
	);
	const targetBranch = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.targetBranch ?? null
	);
	const startDocGeneration = useDocGenerationStore((s) => s.start);
	const resetDocGeneration = useDocGenerationStore((s) => s.reset);
	const commitDocGeneration = useDocGenerationStore((s) => s.commit);
	const cancelDocGenerationStore = useDocGenerationStore((s) => s.cancel);
	const setActiveTabStore = useDocGenerationStore((s) => s.setActiveTab);
	const setCompletedCommitInfoStore = useDocGenerationStore(
		(s) => s.setCompletedCommitInfo
	);
	const prevDocResultRef = useRef(docResult);
	const prevStatusRef = useRef(status);

	const isRunning = status === "running";
	const isCommitting = status === "committing";
	const isBusy = isRunning || isCommitting;
	const hasGenerationAttempt =
		status !== "idle" || Boolean(docResult) || events.length > 0;

	const setActiveTab = useCallback(
		(tab: "activity" | "review" | "summary") => {
			setActiveTabStore(projectKey, tab);
		},
		[projectKey, setActiveTabStore]
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
		resetDocGeneration(projectKey);
	}, [projectKey, resetDocGeneration]);

	const setCompletedCommit = useCallback(
		(newSourceBranch: string, newTargetBranch: string) => {
			setCompletedCommitInfoStore(projectKey, {
				sourceBranch: newSourceBranch,
				targetBranch: newTargetBranch,
			});
		},
		[projectKey, setCompletedCommitInfoStore]
	);

	const cancelDocGeneration = useCallback(() => {
		cancelDocGenerationStore(projectKey);
	}, [cancelDocGenerationStore, projectKey]);

	return {
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
		isBusy,
		hasGenerationAttempt,
		startDocGeneration,
		commitDocGeneration,
		cancelDocGeneration,
		reset,
		setCompletedCommit,
	};
};
