import type { models } from "@go/models";
import { ListBranchesByPath } from "@go/services/GitService";
import { useCallback, useEffect, useState } from "react";

export const useBranchManager = (repoPath: string | undefined) => {
	const [branches, setBranches] = useState<models.BranchInfo[]>([]);
	const [sourceBranch, setSourceBranch] = useState<string | undefined>();
	const [targetBranch, setTargetBranch] = useState<string | undefined>();
	const [sourceOpen, setSourceOpen] = useState(false);
	const [targetOpen, setTargetOpen] = useState(false);

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

	const swapBranches = useCallback(() => {
		setSourceBranch((currentSource) => {
			const next = targetBranch;
			setTargetBranch(currentSource);
			return next;
		});
	}, [targetBranch]);

	const resetBranches = useCallback(() => {
		setSourceBranch(undefined);
		setTargetBranch(undefined);
		setSourceOpen(false);
		setTargetOpen(false);
	}, []);

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
	};
};
