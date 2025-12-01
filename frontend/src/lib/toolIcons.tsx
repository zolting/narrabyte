import {
	FileText,
	FilePlus,
	FileEdit,
	FolderOpen,
	Search,
	SearchCode,
	Terminal,
	Trash2,
	Move,
	Copy,
	Wrench,
	type LucideIcon,
} from "lucide-react";

export type ToolType =
	| "read"
	| "write"
	| "edit"
	| "list"
	| "glob"
	| "grep"
	| "bash"
	| "delete"
	| "move"
	| "copy"
	| "unknown";

export const toolIconMap: Record<ToolType, LucideIcon> = {
	read: FileText,
	write: FilePlus,
	edit: FileEdit,
	list: FolderOpen,
	glob: Search,
	grep: SearchCode,
	bash: Terminal,
	delete: Trash2,
	move: Move,
	copy: Copy,
	unknown: Wrench,
};

export function getToolIcon(toolType: string): LucideIcon {
	const normalizedType = toolType.toLowerCase() as ToolType;
	return toolIconMap[normalizedType] || toolIconMap.unknown;
}
