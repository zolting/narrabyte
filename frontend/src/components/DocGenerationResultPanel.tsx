import type { models } from "@go/models";
import { useEffect, useMemo, useRef, useState } from "react";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { DocGenerationStatus } from "@/stores/docGeneration";
import "react-diff-view/style/index.css";
import "./GitDiffDialog/diff-view-theme.css";

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
        status,
        isBusy,
        onRequestChanges,
}: {
        result: models.DocGenerationResult | null;
        status: DocGenerationStatus;
        isBusy: boolean;
        onRequestChanges: (message: string) => Promise<void>;
}) {
        const { t } = useTranslation();
        const [feedback, setFeedback] = useState("");
        const [viewType, setViewType] = useState<"split" | "unified">("unified");
        const conversationContainerRef = useRef<HTMLDivElement | null>(null);
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

        const conversationEntries = useMemo(() => result?.conversation ?? [], [result?.conversation]);

        useEffect(() => {
                const container = conversationContainerRef.current;
                if (container) {
                        container.scrollTop = container.scrollHeight;
                }
        }, [conversationEntries.length]);

        const disableInput = isBusy;

        const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
                event.preventDefault();
                const trimmed = feedback.trim();
                if (trimmed.length === 0) {
                        return;
                }
                try {
                        await onRequestChanges(trimmed);
                        setFeedback("");
                } catch (error) {
                        console.error("Failed to submit documentation feedback", error);
                }
        };

        if (!result) {
                return null;
        }

        const hasDiff = entries.length > 0 && result.diff.trim().length > 0;

        const roleLabel = (role: string) => {
                switch (role) {
                        case "assistant":
                                return t("common.assistant", "Assistant");
                        case "user":
                                return t("common.you", "You");
                        case "context":
                                return t("common.contextProvided", "Context provided to the assistant");
                        default:
                                return role;
                }
        };

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
                        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(280px,0.35fr)]">
                                <div className="flex min-h-0 flex-col gap-4 overflow-hidden">
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
                                <div className="flex min-h-0 flex-col gap-3 overflow-hidden rounded-md border border-border bg-muted/40 p-4">
                                        <div className="flex items-center justify-between gap-2">
                                                <h3 className="font-semibold text-foreground text-sm uppercase tracking-wide">
                                                        {t("common.conversation", "Conversation")}
                                                </h3>
                                                {status === "running" && (
                                                        <span className="text-muted-foreground text-xs">
                                                                {t(
                                                                        "common.awaitingResponse",
                                                                        "Awaiting assistant response…"
                                                                )}
                                                        </span>
                                                )}
                                        </div>
                                        <div
                                                className="flex-1 space-y-3 overflow-y-auto pr-1"
                                                ref={conversationContainerRef}
                                        >
                                                {conversationEntries.length === 0 ? (
                                                        <p className="text-muted-foreground text-sm">
                                                                {t(
                                                                        "common.noConversation",
                                                                        "No conversation yet. Ask the assistant to adjust the documentation."
                                                                )}
                                                        </p>
                                                ) : (
                                                        conversationEntries.map((message, index) => {
                                                                const roleClass =
                                                                        message.role === "assistant"
                                                                                ? "bg-card"
                                                                                : message.role === "user"
                                                                                ? "bg-secondary"
                                                                                : "bg-muted";
                                                                return (
                                                                        <div
                                                                                className={cn(
                                                                                        "rounded-md border border-border p-3 text-sm leading-relaxed shadow-xs text-foreground",
                                                                                        roleClass
                                                                                )}
                                                                                key={`${message.role}-${index}`}
                                                                        >
                                                                                <div className="font-semibold text-xs uppercase tracking-wide text-foreground/80">
                                                                                        {roleLabel(message.role)}
                                                                                </div>
                                                                                <div className="mt-2 max-h-48 overflow-y-auto whitespace-pre-wrap break-words text-foreground">
                                                                                        {message.content}
                                                                                </div>
                                                                        </div>
                                                                );
                                                        })
                                                )}
                                        </div>
                                        <form className="space-y-3" onSubmit={handleSubmit}>
                                                <div className="space-y-1">
                                                        <label className="sr-only" htmlFor="doc-feedback">
                                                                {t(
                                                                        "common.feedbackPlaceholder",
                                                                        "Describe the changes you would like"
                                                                )}
                                                        </label>
                                                        <textarea
                                                                className="min-h-[5.5rem] w-full rounded-md border border-border bg-background p-3 text-foreground text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-primary disabled:cursor-not-allowed disabled:opacity-60"
                                                                disabled={disableInput}
                                                                id="doc-feedback"
                                                                onChange={(event) => setFeedback(event.target.value)}
                                                                placeholder={t(
                                                                        "common.feedbackPlaceholder",
                                                                        "Describe the adjustments you would like the assistant to make"
                                                                )}
                                                                value={feedback}
                                                        />
                                                </div>
                                                <div className="flex justify-end">
                                                        <Button disabled={disableInput || feedback.trim().length === 0} type="submit">
                                                                {t("common.send", "Send")}
                                                        </Button>
                                                </div>
                                        </form>
                                </div>
                        </div>
                </section>
        );
}
