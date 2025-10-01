import type { models } from "@go/models";
import { useEffect, useMemo, useState } from "react";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import { useTranslation } from "react-i18next";
import { DocRefinementChat } from "@/components/DocRefinementChat";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import "react-diff-view/style/index.css";
import "./GitDiffDialog/diff-view-theme.css";
import { useDocGenerationStore } from "@/stores/docGeneration";

const STARTS_WITH_A_SLASH_REGEX = /^a\//;
const STARTS_WITH_B_SLASH_REGEX = /^b\//;

function normalizeDiffPath(path?: string | null): string {
	if (!path) {
		return "";
	}
	return path
		.replace(STARTS_WITH_A_SLASH_REGEX, "")
		.replace(STARTS_WITH_B_SLASH_REGEX, "");
}

const statusClassMap: Record<string, string> = {
	added: "text-emerald-600",
	modified: "text-blue-600",
	deleted: "text-red-600",
	renamed: "text-amber-600",
	copied: "text-purple-600",
	untracked: "text-emerald-600",
};

export function DocGenerationResultPanel({
	result,
	projectId,
}: {
	result: models.DocGenerationResult | null;
	projectId: number;
}) {
	const toggleChat = useDocGenerationStore((s) => s.toggleChat);
	const chatOpen = useDocGenerationStore((s) => s.chatOpen);
	const { t } = useTranslation();
	const [viewType, setViewType] = useState<"split" | "unified">("unified");

	const parsedDiff = useMemo(() => {
		if (!result?.diff) {
			return [];
		}
		try {
			return parseDiff(result.diff);
		} catch (error) {
			console.error("Failed to parse documentation diff", error);
			return [];
		}
	}, [result?.diff]);

	const statusMap = useMemo(() => {
		const map = new Map<string, string>();
		if (result?.files) {
			for (const file of result.files) {
				map.set(normalizeDiffPath(file.path), file.status);
			}
		}
		return map;
	}, [result?.files]);

	const entries = useMemo(
		() =>
			parsedDiff.map((file) => {
				const key = normalizeDiffPath(
					file.newPath && file.newPath !== "/dev/null"
						? file.newPath
						: file.oldPath
				);
				return {
					diff: file,
					path: key,
					status: statusMap.get(key) ?? "changed",
				};
			}),
		[parsedDiff, statusMap]
	);

	const [selectedPath, setSelectedPath] = useState<string | null>(null);

	useEffect(() => {
		if (!result) {
			setSelectedPath(null);
			return;
		}
		const firstPath = entries[0]?.path ?? result.files?.[0]?.path ?? null;
		setSelectedPath(firstPath ? normalizeDiffPath(firstPath) : null);
	}, [entries, result]);

	const activeEntry = useMemo(
		() => entries.find((entry) => entry.path === selectedPath),
		[entries, selectedPath]
	);

	if (!result) {
		return null;
	}

	const hasDiff = entries.length > 0 && result.diff.trim().length > 0;

	// When chat is closed, don't allocate a grid column for it.
	const layoutClass = cn(
		"flex min-h-0 flex-1 flex-col gap-4 overflow-hidden",
		chatOpen && "lg:grid lg:grid-cols-[1fr_360px]"
	);

	return (
		<section className="flex h-full flex-col gap-4 overflow-hidden rounded-lg border border-border bg-card p-4">
			<header className="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
				<div>
					<h2 className="font-semibold text-foreground text-lg">
						{t("common.documentationUpdates", "Documentation Updates")}
					</h2>
					<p className="text-muted-foreground text-sm">
						{t("common.branch", "Branch")}: {result.branch}
					</p>
				</div>
				<div className="space-x-2">
					{hasDiff && (
						<Button
							className="border-border text-foreground hover:bg-accent"
							onClick={() =>
								setViewType((prev) => (prev === "split" ? "unified" : "split"))
							}
							size="sm"
							variant="outline"
						>
							{viewType === "split"
								? t("common.inlineView", "Inline view")
								: t("common.splitView", "Split view")}
						</Button>
					)}

					<Button
						className="border-border"
						onClick={() => toggleChat()}
						size="sm"
						variant="outline"
					>
						{chatOpen
							? t("common.hideChat", "Hide chat")
							: t("common.showChat", "Show chat")}
					</Button>
				</div>
			</header>

			<div className={layoutClass}>
				<div className="min-h-0 flex-1 overflow-hidden">
					{hasDiff ? (
						<div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden lg:grid lg:grid-cols-[220px_1fr]">
							<div className="flex max-h-48 min-h-0 flex-col gap-2 overflow-hidden lg:h-full lg:max-h-none">
								<div className="text-muted-foreground text-xs uppercase tracking-wide">
									{t("common.files", "Files")}
								</div>
								<ul className="min-h-0 flex-1 space-y-1 overflow-y-auto pr-1">
									{entries.map((entry) => (
										<li key={entry.path}>
											<button
												className={cn(
													"w-full rounded-md border border-transparent px-3 py-2 text-left transition-colors",
													selectedPath === entry.path
														? "bg-accent text-accent-foreground"
														: "hover:bg-muted"
												)}
												onClick={() => setSelectedPath(entry.path)}
												type="button"
											>
												<div
													className={cn(
														"font-medium text-xs",
														statusClassMap[entry.status.toLowerCase()] ??
															"text-foreground/70"
													)}
												>
													{entry.status}
												</div>
												<div className="truncate font-mono text-foreground/90 text-sm">
													{entry.path}
												</div>
											</button>
										</li>
									))}
								</ul>
							</div>

							<div className="min-h-0 flex-1 overflow-hidden rounded-md border border-border text-xs">
								<div className="h-full overflow-y-auto">
									{activeEntry ? (
										<Diff
											className="text-foreground"
											diffType={activeEntry.diff.type}
											hunks={activeEntry.diff.hunks}
											optimizeSelection={false}
											viewType={viewType}
										>
											{(hunks) =>
												hunks.map((hunk) => (
													<Hunk hunk={hunk} key={hunk.content} />
												))
											}
										</Diff>
									) : (
										<div className="p-4 text-muted-foreground text-sm">
											{t(
												"common.selectFile",
												"Select a file to preview the diff."
											)}
										</div>
									)}
								</div>
							</div>
						</div>
					) : (
						<div className="rounded-md border border-border border-dashed p-4 text-muted-foreground text-sm">
							{t(
								"common.noDocumentationChanges",
								"No documentation changes were produced for this diff."
							)}
						</div>
					)}
				</div>

				{chatOpen && (
					<div className="min-h-0">
						<DocRefinementChat branch={result.branch} projectId={projectId} />
					</div>
				)}
			</div>
		</section>
	);
}
