import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { DemoEvent } from "@/types/events";
import { useDocGenerationStore } from "@/stores/docGeneration";

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
	const startDocGeneration = useDocGenerationStore((s) => s.start);
	const resetDocGeneration = useDocGenerationStore((s) => s.reset);
	const commitDocGeneration = useDocGenerationStore((s) => s.commit);
	const cancelDocGenerationStore = useDocGenerationStore((s) => s.cancel);

	const [activeTab, setActiveTab] = useState<"activity" | "review" | "summary">(
		"activity"
	);
	const [commitCompleted, setCommitCompleted] = useState(false);
	const [completedCommitInfo, setCompletedCommitInfo] = useState<{
		sourceBranch: string;
		targetBranch: string;
	} | null>(null);

	const isRunning = status === "running";
	const isCommitting = status === "committing";
	const isBusy = isRunning || isCommitting;
	const hasGenerationAttempt =
		status !== "idle" || Boolean(docResult) || events.length > 0;

	// Switch to review tab when LLM completes
	useEffect(() => {
		if (docResult) {
			setActiveTab("review");
		}
	}, [docResult]);

	// Handle tab switching during status changes
	useEffect(() => {
		if (status === "running" || status === "committing") {
			setActiveTab("activity");
		}
		if (status === "success" && commitCompleted) {
			return; // Keep current tab when commit completes
		}
	}, [status, commitCompleted]);

	// Detect successful commit completion
	const prevStatusRef = useRef(status);
	useEffect(() => {
		if (
			prevStatusRef.current === "committing" &&
			status === "success" &&
			docResult
		) {
			setCommitCompleted(true);
		}
		prevStatusRef.current = status;
	}, [status, docResult]);

	const reset = useCallback(() => {
		resetDocGeneration(projectKey);
		setActiveTab("activity");
		setCommitCompleted(false);
		setCompletedCommitInfo(null);
	}, [projectKey, resetDocGeneration]);

	const setCompletedCommit = useCallback(
		(sourceBranch: string, targetBranch: string) => {
			setCompletedCommitInfo({ sourceBranch, targetBranch });
		},
		[]
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
