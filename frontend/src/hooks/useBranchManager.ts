import type { models } from "@go/models";
import { ListBranchesByPath } from "@go/services/GitService";
import { useCallback, useEffect, useState } from "react";
import { sortBranches } from "@/lib/sortBranches";

/**
 * Hook for fetching and managing the list of branches (shared state).
 * Does NOT manage selection state - that should be per-tab.
 */
export const useBranchList = (repoPath: string | undefined) => {
	const [branches, setBranches] = useState<models.BranchInfo[]>([]);

	const fetchBranches = useCallback(() => {
		if (!repoPath) {
			setBranches([]);
			return;
		}

		ListBranchesByPath(repoPath)
			.then((arr) => {
				setBranches(sortBranches(arr));
			})
			.catch((err) => console.error("failed to fetch branches:", err));
	}, [repoPath]);

	// Fetch branches when repoPath changes
	useEffect(() => {
		if (!repoPath) {
			setBranches([]);
			return;
		}

		let isActive = true;
		ListBranchesByPath(repoPath)
			.then((arr) => {
				if (!isActive) {
					return;
				}
				setBranches(sortBranches(arr));
			})
			.catch((err) => console.error("failed to fetch branches:", err));

		return () => {
			isActive = false;
		};
	}, [repoPath]);

	return {
		branches,
		fetchBranches,
	};
};

/**
 * Hook for managing branch selection state (per-tab state).
 * Each tab should have its own instance of this hook.
 */
export const useBranchSelection = () => {
	const [sourceBranch, setSourceBranch] = useState<string | undefined>();
	const [targetBranch, setTargetBranch] = useState<string | undefined>();
	const [sourceOpen, setSourceOpen] = useState(false);
	const [targetOpen, setTargetOpen] = useState(false);

	const resetSelection = useCallback(() => {
		setSourceBranch(undefined);
		setTargetBranch(undefined);
		setSourceOpen(false);
		setTargetOpen(false);
	}, []);

	const swapBranches = useCallback(() => {
		setSourceBranch((currentSource) => {
			const next = targetBranch;
			setTargetBranch(currentSource);
			return next;
		});
	}, [targetBranch]);

	return {
		sourceBranch,
		setSourceBranch,
		targetBranch,
		setTargetBranch,
		sourceOpen,
		setSourceOpen,
		targetOpen,
		setTargetOpen,
		swapBranches,
		resetSelection,
	};
};

/**
 * Combined hook for backward compatibility.
 * @deprecated Use useBranchList + useBranchSelection separately for per-tab state.
 */
export const useBranchManager = (repoPath: string | undefined) => {
	const { branches, fetchBranches } = useBranchList(repoPath);
	const {
		sourceBranch,
		setSourceBranch,
		targetBranch,
		setTargetBranch,
		sourceOpen,
		setSourceOpen,
		targetOpen,
		setTargetOpen,
		swapBranches,
		resetSelection,
	} = useBranchSelection();

	const resetBranches = useCallback(() => {
		resetSelection();
	}, [resetSelection]);

	return {
		branches,
		sourceBranch,
		setSourceBranch,
		targetBranch,
		setTargetBranch,
		sourceOpen,
		setSourceOpen,
		targetOpen,
		setTargetOpen,
		swapBranches,
		resetBranches,
		fetchBranches,
	};
};
