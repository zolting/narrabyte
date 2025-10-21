import type { models } from "@go/models";

const PRIORITY_BRANCHES = ["main", "master", "develop"] as const;

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

	const sortedByRecency = branches.sort(compareByLastCommit);
	if (!prioritizeMainMaster) {
		return sortedByRecency;
	}

	const prioritized: models.BranchInfo[] = [];
	const remaining: models.BranchInfo[] = [];

	for (const branch of sortedByRecency) {
		const name = branchName(branch);
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
	}

	prioritized.push(...remaining);

	return prioritized;
}
