import { useCallback, useEffect, useRef, useState } from "react";
import { useDocGenerationStore } from "@/stores/docGeneration";

export const useDocGenerationManager = () => {
	const docResult = useDocGenerationStore((s) => s.result);
	const status = useDocGenerationStore((s) => s.status);
	const events = useDocGenerationStore((s) => s.events);
	const startDocGeneration = useDocGenerationStore((s) => s.start);
	const resetDocGeneration = useDocGenerationStore((s) => s.reset);
	const commitDocGeneration = useDocGenerationStore((s) => s.commit);
	const cancelDocGeneration = useDocGenerationStore((s) => s.cancel);
	const requestDocChanges = useDocGenerationStore((s) => s.requestChanges);
	const docGenerationError = useDocGenerationStore((s) => s.error);
	const pendingUserMessage = useDocGenerationStore((s) => s.pendingUserMessage);

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
		resetDocGeneration();
		setActiveTab("activity");
		setCommitCompleted(false);
		setCompletedCommitInfo(null);
	}, [resetDocGeneration]);

	const setCompletedCommit = useCallback(
		(sourceBranch: string, targetBranch: string) => {
			setCompletedCommitInfo({ sourceBranch, targetBranch });
		},
		[]
	);

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
		requestDocChanges,
		reset,
		setCompletedCommit,
		pendingUserMessage,
	};
};
