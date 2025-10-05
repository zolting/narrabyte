import type { models } from "@go/models";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import { useTranslation } from "react-i18next";
import { DocRefinementChat } from "@/components/DocRefinementChat";
import { Button } from "@/components/ui/button";
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "@/components/ui/tooltip";
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
	const projectKey = useMemo(() => String(projectId), [projectId]);
	const toggleChatStore = useDocGenerationStore((s) => s.toggleChat);
	const chatOpen = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.chatOpen ?? false
	);
	const changedSinceInitial = useDocGenerationStore(
		(s) => s.docStates[projectKey]?.changedSinceInitial ?? []
	);
	const { t } = useTranslation();
	const [viewType, setViewType] = useState<"split" | "unified">("unified");

	const handleToggleChat = useCallback(() => {
		toggleChatStore(projectKey);
	}, [toggleChatStore, projectKey]);

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

	return (
		<section className="flex h-full flex-col gap-4 overflow-hidden rounded-lg border border-border bg-card p-4">
			<header className="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
				<div>
					<h2 className="font-semibold text-foreground text-lg">
						{t("common.documentationUpdates", "Documentation Updates")}
					</h2>
					<div className="text-muted-foreground text-xs sm:text-sm space-y-1">
						<p>
							{t("common.sourceBranch", "Source branch")}: {result.branch}
						</p>
						{result.docsBranch && (
							<p>
								{t("common.docsBranch", "Documentation branch")}: {result.docsBranch}
							</p>
						)}
						{result.docsInCodeRepo && (
							<p className="text-emerald-600">
								{t(
									"common.docsSharedWithCode",
									"Documentation changes were generated directly inside the code repository"
								)}
							</p>
						)}
					</div>
				</div>
				<div>
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
						className="ml-2 border-border text-foreground hover:bg-accent"
						onClick={handleToggleChat}
						size="sm"
						variant="outline"
					>
						{chatOpen
							? t("common.hideChat", "Hide chat")
							: t("common.showChat", "Show chat")}
					</Button>
				</div>
			</header>

			{hasDiff ? (
				<div
					className={cn(
						"flex min-h-0 flex-1 flex-col gap-4 overflow-hidden lg:grid",
						chatOpen
							? "lg:grid-cols-[220px_1fr_360px]" // files | diff | chat
							: "lg:grid-cols-[220px_1fr]" // files | diff
					)}
				>
					<div className="flex max-h-48 min-h-0 flex-col gap-2 overflow-hidden lg:h-full lg:max-h-none">
						<div className="text-muted-foreground text-xs uppercase tracking-wide">
							{t("common.files", "Files")}
						</div>
						<ul className="min-h-0 flex-1 space-y-0.5 overflow-y-auto pr-1">
							{entries.map((entry) => (
								<li key={entry.path}>
									<TooltipProvider delayDuration={500}>
										<Tooltip>
											<TooltipTrigger asChild>
												<button
													className={cn(
														"group w-full rounded-md border border-transparent px-2 py-1.5 text-left transition-colors",
														selectedPath === entry.path
															? "bg-accent text-accent-foreground"
															: "hover:bg-muted"
													)}
													onClick={() => setSelectedPath(entry.path)}
													type="button"
												>
													{(() => {
														const isChanged = (
															changedSinceInitial || []
														).includes(entry.path);
														return (
															<div>
																<div
																	className={cn(
																		"font-medium text-[11px]",
																		statusClassMap[
																			entry.status.toLowerCase()
																		] ?? "text-foreground/70"
																	)}
																>
																	{entry.status}
																	{isChanged && (
																		<span className="ml-2 inline-flex items-center rounded border border-amber-200 bg-amber-100/60 px-1.5 py-0.5 font-medium text-[10px] text-amber-800">
																			{t("common.updated")}
																		</span>
																	)}
																</div>
																<div className="truncate font-mono text-foreground/90 text-xs transition-colors group-hover:text-foreground">
																	{entry.path}
																</div>
															</div>
														);
													})()}
												</button>
											</TooltipTrigger>
											<TooltipContent side="right">
												<p className="max-w-md break-all font-mono text-xs">
													{entry.path}
												</p>
											</TooltipContent>
										</Tooltip>
									</TooltipProvider>
								</li>
							))}
						</ul>
					</div>

					<div className="min-h-0 flex-1 overflow-hidden rounded-md border border-border text-xs">
						<div className="h-full overflow-auto">
							{activeEntry ? (
								<Diff
									className="text-foreground"
									diffType={activeEntry.diff.type}
									hunks={activeEntry.diff.hunks}
									optimizeSelection={false}
									viewType={viewType}
								>
									{(hunks) =>
										hunks.map((hunk) => <Hunk hunk={hunk} key={hunk.content} />)
									}
								</Diff>
							) : (
								<div className="p-4 text-muted-foreground text-sm">
									{t("common.selectFile", "Select a file to preview the diff.")}
								</div>
							)}
						</div>
					</div>

					{chatOpen && (
						<div className="h-full min-h-0 overflow-hidden">
							<DocRefinementChat branch={result.branch} projectId={projectId} />
						</div>
					)}
				</div>
			) : (
				<div className="rounded-md border border-border border-dashed p-4 text-muted-foreground text-sm">
					{t(
						"common.noDocumentationChanges",
						"No documentation changes were produced for this diff."
					)}
				</div>
			)}
		</section>
	);
}
