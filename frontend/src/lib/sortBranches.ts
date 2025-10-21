import type { models } from "@go/models";

const PRIORITY_BRANCHES = ["main", "master"] as const;

const priorityOrder = new Map<string, number>(
	PRIORITY_BRANCHES.map((branch, index) => [branch, index])
);

const branchName = (branch: models.BranchInfo) => (branch.name ?? "").trim();

const compareByLastCommit = (a: models.BranchInfo, b: models.BranchInfo) => {
	const dateA = new Date(a.lastCommitDate as unknown as string).getTime();
	const dateB = new Date(b.lastCommitDate as unknown as string).getTime();
	return dateB - dateA;
};

export interface SortBranchesOptions {
	prioritizeMainMaster?: boolean;
}

/**
 * Returns a new array of branches sorted by most recent commit, optionally
 * ensuring `main` and `master` appear first (in that order when present).
 */
export function sortBranches(
	branches: models.BranchInfo[],
	{ prioritizeMainMaster = false }: SortBranchesOptions = {}
): models.BranchInfo[] {
	if (!Array.isArray(branches) || branches.length === 0) {
		return [];
	}

	const sortedByRecency = [...branches].sort(compareByLastCommit);
	if (!prioritizeMainMaster) {
		return sortedByRecency;
	}

	const prioritized: models.BranchInfo[] = [];
	const seen = new Set<string>();
	const remaining: models.BranchInfo[] = [];

	for (const branch of sortedByRecency) {
		const name = branchName(branch);
		if (!name || seen.has(name)) {
			continue;
		}
		const priorityIndex = priorityOrder.get(name);
		if (priorityIndex === undefined) {
			remaining.push(branch);
			continue;
		}
		// Priority order: main (index 0) before master.
		if (priorityIndex === 0) {
			prioritized.unshift(branch);
		} else {
			prioritized.push(branch);
		}
		seen.add(name);
	}

	for (const branch of remaining) {
		const name = branchName(branch);
		if (!seen.has(name)) {
			prioritized.push(branch);
			seen.add(name);
		}
	}

	return prioritized;
}
