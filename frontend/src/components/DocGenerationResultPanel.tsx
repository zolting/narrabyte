import { useEffect, useMemo, useState } from "react";
import type { models } from "@go/models";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import "react-diff-view/style/index.css";
import "./GitDiffDialog/diff-view-theme.css";

function normalizeDiffPath(path?: string | null): string {
	if (!path) {
		return "";
	}
	return path.replace(/^a\//, "").replace(/^b\//, "");
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
}: {
	result: models.DocGenerationResult | null;
}) {
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
		result?.files?.forEach((file: models.DocChangedFile) => {
			map.set(normalizeDiffPath(file.path), file.status);
		});
		return map;
	}, [result?.files]);

	const entries = useMemo(
		() =>
			parsedDiff.map((file) => {
				const key = normalizeDiffPath(
					file.newPath && file.newPath !== "/dev/null"
						? file.newPath
						: file.oldPath,
				);
				return {
					diff: file,
					path: key,
					status: statusMap.get(key) ?? "changed",
				};
			}),
		[parsedDiff, statusMap],
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
		[entries, selectedPath],
	);

	if (!result) {
		return null;
	}

	const hasDiff = entries.length > 0 && result.diff.trim().length > 0;

	return (
		<section className="space-y-4 rounded-lg border border-border bg-card p-4">
			<header className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
				<div>
					<h2 className="font-semibold text-foreground text-lg">
						{t("common.documentationUpdates", "Documentation Updates")}
					</h2>
						<p className="text-muted-foreground text-sm">
							{t("common.branch", "Branch")}: {result.branch}
					</p>
				</div>
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
			</header>
				{result.summary && (
					<p className="rounded-md border border-border bg-muted/40 p-3 text-sm text-foreground/90">
						{result.summary}
					</p>
				)}
			{hasDiff ? (
				<div className="grid gap-4 lg:grid-cols-[220px_1fr]">
					<div className="space-y-2">
						<div className="text-muted-foreground text-xs uppercase tracking-wide">
							{t("common.files", "Files")}
						</div>
						<ul className="space-y-1">
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
												"text-xs font-medium",
												statusClassMap[entry.status.toLowerCase()] ?? "text-foreground/70",
											)}
										>
											{entry.status}
										</div>
										<div className="truncate text-sm font-mono text-foreground/90">
											{entry.path}
										</div>
									</button>
								</li>
							))}
						</ul>
					</div>
					<div className="overflow-hidden rounded-md border border-border">
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
							<div className="p-4 text-sm text-muted-foreground">
								{t("common.selectFile", "Select a file to preview the diff.")}
							</div>
						)}
					</div>
				</div>
			) : (
				<div className="rounded-md border border-dashed border-border p-4 text-sm text-muted-foreground">
					{t("common.noDocumentationChanges", "No documentation changes were produced for this diff.")}
				</div>
			)}
		</section>
	);
}
