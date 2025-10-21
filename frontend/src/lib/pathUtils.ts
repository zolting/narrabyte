const normalize = (input: string) => {
	if (!input) {
		return "";
	}
	return input.replace(/\\/g, "/").replace(/\/+$/u, "").trim();
};

const isSubPath = (possibleChild: string, possibleParent: string) => {
	if (!possibleChild || !possibleParent) {
		return false;
	}
	if (possibleChild === possibleParent) {
		return true;
	}
	return possibleChild.startsWith(`${possibleParent}/`);
};

export const pathsShareRoot = (a?: string, b?: string) => {
	const normA = normalize(a ?? "");
	const normB = normalize(b ?? "");
	if (!normA || !normB) {
		return false;
	}
	return isSubPath(normA, normB) || isSubPath(normB, normA);
};

