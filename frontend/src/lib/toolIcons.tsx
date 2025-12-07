import {
	BookOpen,
	CheckSquare,
	Code,
	Copy,
	FileEdit,
	FilePlus,
	FileText,
	FolderOpen,
	ListTodo,
	type LucideIcon,
	Move,
	Search,
	SearchCode,
	Terminal,
	Trash2,
	Wrench,
} from "lucide-react";

export type ToolType =
	| "read"
	| "write"
	| "multiedit"
	| "edit"
	| "list"
	| "glob"
	| "grep"
	| "bash"
	| "delete"
	| "move"
	| "copy"
	| "todo_read"
	| "todo_write"
	| "unknown";

export const toolIconMap: Record<ToolType, LucideIcon> = {
	read: FileText,
	write: FilePlus,
	edit: FileEdit,
	multiedit: FileEdit,
	list: FolderOpen,
	glob: Search,
	grep: SearchCode,
	bash: Terminal,
	delete: Trash2,
	move: Move,
	copy: Copy,
	todo_read: ListTodo,
	todo_write: CheckSquare,
	unknown: Wrench,
};

export function getToolIcon(toolType: string): LucideIcon {
	const normalizedType = toolType.toLowerCase() as ToolType;
	return toolIconMap[normalizedType] || toolIconMap.unknown;
}

/**
 * Returns an icon based on the path prefix (docs: or code:)
 * @param path - The file path that may be prefixed with "docs:" or "code:"
 * @returns BookOpen for docs paths, Code for code paths, null otherwise
 */
export function getPathPrefixIcon(path: string): LucideIcon | null {
	if (!path) return null;

	if (path.startsWith("docs:")) {
		return BookOpen;
	}

	if (path.startsWith("code:")) {
		return Code;
	}

	return null;
}

/**
 * Strips the "docs:" or "code:" prefix from a path
 * @param path - The file path that may be prefixed
 * @returns The path without the prefix
 */
export function stripPathPrefix(path: string): string {
	if (!path) return path;

	if (path.startsWith("docs:")) {
		return path.substring(5); // Remove "docs:"
	}

	if (path.startsWith("code:")) {
		return path.substring(5); // Remove "code:"
	}

	return path;
}
