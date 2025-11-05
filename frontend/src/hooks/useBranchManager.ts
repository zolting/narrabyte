import type { models } from "@go/models";
import { ListBranchesByPath } from "@go/services/GitService";
import { useCallback, useEffect, useState } from "react";
import { sortBranches } from "@/lib/sortBranches";

export const useBranchManager = (repoPath: string | undefined) => {
	const [branches, setBranches] = useState<models.BranchInfo[]>([]);
	const [sourceBranch, setSourceBranch] = useState<string | undefined>();
	const [targetBranch, setTargetBranch] = useState<string | undefined>();
	const [sourceOpen, setSourceOpen] = useState(false);
	const [targetOpen, setTargetOpen] = useState(false);

	const resetBranches = useCallback(() => {
		setSourceBranch(undefined);
		setTargetBranch(undefined);
		setSourceOpen(false);
		setTargetOpen(false);
	}, []);

	const fetchBranches = useCallback(() => {
		if (!repoPath) {
			setBranches([]);
			setSourceBranch(undefined);
			setTargetBranch(undefined);
			return;
		}

		ListBranchesByPath(repoPath)
			.then((arr) => {
				setBranches(sortBranches(arr));
			})
			.catch((err) => console.error("failed to fetch branches:", err));
	}, [repoPath]);

	const swapBranches = useCallback(() => {
		setSourceBranch((currentSource) => {
			const next = targetBranch;
			setTargetBranch(currentSource);
			return next;
		});
	}, [targetBranch]);

	// Fetch branches when repoPath changes
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
				setBranches(sortBranches(arr));
			})
			.catch((err) => console.error("failed to fetch branches:", err));

		return () => {
			isActive = false;
		};
	}, [repoPath]);

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
