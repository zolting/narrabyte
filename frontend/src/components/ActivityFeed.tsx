import {
	CheckCircle2,
	ChevronDown,
	ChevronUp,
	Circle,
	Loader2,
	XCircle,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { MarkdownRenderer } from "@/components/MarkdownRenderer";
import {
	getPathPrefixIcon,
	stripPathPrefix,
	type ToolType,
} from "@/lib/toolIcons";
import { cn } from "@/lib/utils";
import type { DocGenerationStatus } from "@/stores/docGeneration";
import type { TodoItem, ToolEvent } from "@/types/events";

const REASONING_STREAM = "reasoning";
const STREAM_METADATA_KEY = "stream";
const STREAM_STATE_KEY = "state";
const STREAM_STATE_RESET = "reset";
const STREAM_STATE_UPDATE = "update";

export function ActivityFeed({
	events,
	todos,
	status,
}: {
	events: ToolEvent[];
	todos: TodoItem[];
	status: DocGenerationStatus;
}) {
	const { t } = useTranslation();
	const containerRef = useRef<HTMLDivElement | null>(null);
	const previousEventCountRef = useRef(0);
	const [visibleEvents, setVisibleEvents] = useState<string[]>([]);
	const [showAllTodos, setShowAllTodos] = useState(false);
	const [expandedReasoning, setExpandedReasoning] = useState<Set<string>>(
		new Set()
	);

	const displayEvents = useMemo(() => {
		const result: ToolEvent[] = [];
		const reasoningBlocks = new Map<string, ToolEvent>();
		let currentReasoningId: string | null = null;

		for (const event of events) {
			const streamName = event.metadata?.[STREAM_METADATA_KEY];
			if (streamName === REASONING_STREAM) {
				const state = event.metadata?.[STREAM_STATE_KEY];
				if (state === STREAM_STATE_RESET) {
					// Start a new reasoning block
					currentReasoningId = `reasoning-${event.id}`;
					const reasoningEvent: ToolEvent = {
						id: currentReasoningId,
						type: "info",
						message: "",
						timestamp: event.timestamp,
						metadata: { isReasoning: "true" },
					};
					reasoningBlocks.set(currentReasoningId, reasoningEvent);
					result.push(reasoningEvent);
				} else if (state === STREAM_STATE_UPDATE && currentReasoningId) {
					// Update the current reasoning block
					const existingBlock = reasoningBlocks.get(currentReasoningId);
					if (existingBlock) {
						existingBlock.message = event.message;
						existingBlock.timestamp = event.timestamp;
					}
				}
				continue;
			}
			result.push(event);
		}

		return result;
	}, [events]);

	// Animate new events appearing
	useEffect(() => {
		if (previousEventCountRef.current > displayEvents.length) {
			previousEventCountRef.current = displayEvents.length;
		}
		const newEvents = displayEvents.slice(previousEventCountRef.current);
		previousEventCountRef.current = displayEvents.length;

		if (newEvents.length === 0) {
			return;
		}

		const timeouts = newEvents.map((event, index) =>
			window.setTimeout(() => {
				setVisibleEvents((prev) => {
					if (prev.includes(event.id)) {
						return prev;
					}
					return [...prev, event.id];
				});
			}, index * 100)
		);

		return () => {
			for (const timeout of timeouts) {
				window.clearTimeout(timeout);
			}
		};
	}, [displayEvents]);

	// Reset visible events when list changes
	useEffect(() => {
		setVisibleEvents(displayEvents.map((e) => e.id));
	}, [displayEvents]);

	// Auto-scroll to bottom when new events arrive
	useEffect(() => {
		if (displayEvents.length === 0) {
			return;
		}

		const el = containerRef.current;
		if (!el) {
			return;
		}
		const scrollToBottom = () => {
			el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
		};
		const frameId = window.requestAnimationFrame(scrollToBottom);
		const timeoutId = window.setTimeout(scrollToBottom, 200);
		return () => {
			window.cancelAnimationFrame(frameId);
			window.clearTimeout(timeoutId);
		};
	}, [displayEvents.length]);

	// Find active todo
	const activeTodo = todos.find((todo) => todo.status === "in_progress");

	// Calculate todo counts
	const pendingCount = todos.filter((todo) => todo.status === "pending").length;
	const completedCount = todos.filter(
		(todo) => todo.status === "completed"
	).length;

	// Check if all todos are completed
	const allCompleted = todos.length > 0 && completedCount === todos.length;

	// Get icon for todo status
	const getStatusIcon = (todoStatus: TodoItem["status"]) => {
		switch (todoStatus) {
			case "completed":
				return <CheckCircle2 className="h-4 w-4 text-emerald-600" />;
			case "in_progress":
				return <Loader2 className="h-4 w-4 animate-spin text-blue-600" />;
			case "cancelled":
				return <XCircle className="h-4 w-4 text-muted-foreground" />;
			default:
				return <Circle className="h-4 w-4 text-muted-foreground" />;
		}
	};

	const isRunning = status === "running";
	const isCommitting = status === "committing";
	const inProgress = isRunning || isCommitting;

	const toggleReasoning = (reasoningId: string) => {
		setExpandedReasoning((prev) => {
			const next = new Set(prev);
			if (next.has(reasoningId)) {
				next.delete(reasoningId);
			} else {
				next.add(reasoningId);
			}
			return next;
		});
	};

	// Helper to parse tool events and extract parameters
	const parseToolEvent = (event: ToolEvent) => {
		const toolMetadata = event.metadata?.tool;
		if (!toolMetadata) return null;

		// Map backend tool names to frontend tool types
		const toolNameMap: Record<string, ToolType> = {
			read_file_tool: "read",
			read: "read",
			write_file_tool: "write",
			write: "write",
			edit_file_tool: "edit",
			edit: "edit",
			list_directory_tool: "list",
			list: "list",
			glob_tool: "glob",
			glob: "glob",
			grep_tool: "grep",
			grep: "grep",
			bash_tool: "bash",
			bash: "bash",
			delete_file_tool: "delete",
			delete: "delete",
			move_file_tool: "move",
			move: "move",
			copy_file_tool: "copy",
			copy: "copy",
		};

		const toolType = toolNameMap[toolMetadata];
		if (!toolType) return null;

		// Extract parameters from metadata or message
		const path = event.metadata?.path || "";
		const pattern = event.metadata?.pattern || "";

		// Check for path prefix (docs: or code:) and use appropriate icon
		const prefixIcon = getPathPrefixIcon(path);
		const cleanPath = stripPathPrefix(path);

		return {
			toolType,
			params: { path: cleanPath, pattern },
			prefixIcon,
		};
	};

	// Auto-scroll reasoning blocks to bottom when expanded or content updates
	useEffect(() => {
		const timeouts: number[] = [];
		for (const reasoningId of expandedReasoning) {
			const element = document.querySelector(
				`[data-reasoning-id="${reasoningId}"]`
			);
			if (element) {
				// Scroll to bottom after content is rendered
				// Use setTimeout to ensure DOM is fully updated
				const timeoutId = window.setTimeout(() => {
					element.scrollTo({ top: element.scrollHeight, behavior: "smooth" });
				}, 50);
				timeouts.push(timeoutId);
			}
		}
		return () => {
			for (const timeout of timeouts) {
				window.clearTimeout(timeout);
			}
		};
	}, [expandedReasoning, displayEvents]);

	return (
		<div className="flex min-h-0 flex-1 flex-col gap-2">
			{/* Active todo or completion status - visible when todos exist */}
			{(activeTodo || allCompleted) && (
				<div className="flex flex-col gap-2">
					{activeTodo ? (
						<div className="flex items-center gap-2.5 rounded-md border border-blue-500/30 bg-blue-500/10 px-3 py-2.5">
							<div className="shrink-0">
								<Loader2 className="h-4 w-4 animate-spin text-blue-600" />
							</div>
							<span className="min-w-0 flex-1 break-words font-medium text-foreground text-sm">
								{activeTodo.activeForm}
							</span>
						</div>
					) : (
						<div className="flex items-center gap-2.5 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2.5">
							<div className="shrink-0">
								<CheckCircle2 className="h-4 w-4 text-emerald-600" />
							</div>
							<span className="min-w-0 flex-1 break-words font-medium text-foreground text-sm">
								{t("todos.allCompleted")}
							</span>
						</div>
					)}

					{/* Collapsible todo list */}
					{todos.length > 1 && (
						<div className="overflow-hidden rounded-md border border-border bg-muted/30">
							<button
								className="flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors hover:bg-muted/50"
								onClick={() => setShowAllTodos(!showAllTodos)}
								type="button"
							>
								<span className="font-medium text-foreground">
									{t("activity.allTasks")}
								</span>
								<div className="flex items-center gap-2 text-xs">
									{pendingCount > 0 && (
										<span className="text-muted-foreground">
											{t("todos.pending", {
												count: pendingCount,
											})}
										</span>
									)}
									{completedCount > 0 && (
										<span className="text-emerald-600">
											{t("todos.completed", {
												count: completedCount,
											})}
										</span>
									)}
									{showAllTodos ? (
										<ChevronUp className="h-4 w-4 text-muted-foreground" />
									) : (
										<ChevronDown className="h-4 w-4 text-muted-foreground" />
									)}
								</div>
							</button>

							{showAllTodos && (
								<div className="border-border border-t px-3 py-2">
									<ul className="space-y-1.5">
										{todos.map((todo, index) => {
											const displayText =
												todo.status === "in_progress"
													? todo.activeForm
													: todo.content;

											return (
												<li
													className={cn(
														"flex items-start gap-2.5 rounded-md px-2 py-1.5",
														{
															"bg-emerald-500/10": todo.status === "completed",
															"bg-blue-500/10": todo.status === "in_progress",
															"bg-muted/50": todo.status === "pending",
															"bg-muted/30 opacity-60":
																todo.status === "cancelled",
														}
													)}
													key={`${todo.content}-${todo.status}-${index}`}
												>
													<div className="mt-0.5 shrink-0">
														{getStatusIcon(todo.status)}
													</div>
													<span
														className={cn(
															"min-w-0 flex-1 break-words text-sm",
															{
																"text-foreground": todo.status !== "cancelled",
																"text-muted-foreground line-through":
																	todo.status === "cancelled",
																"font-medium": todo.status === "in_progress",
															}
														)}
													>
														{displayText}
													</span>
												</li>
											);
										})}
									</ul>
								</div>
							)}
						</div>
					)}
				</div>
			)}

			{/* Tool events feed */}
			<div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border border-border bg-muted/30">
				<div className="border-border border-b bg-muted/50 px-3 py-2">
					<div className="flex items-center justify-between">
						<span className="font-medium text-foreground text-sm">
							{(() => {
								if (inProgress) {
									if (isRunning) {
										return t(
											"common.generatingDocs",
											"Generating documentation…"
										);
									}
									return t(
										"common.committingDocs",
										"Committing documentation…"
									);
								}
								return t("activity.toolActivity");
							})()}
						</span>
						{inProgress && (
							<span className="text-muted-foreground text-xs">
								{t("common.inProgress")}
							</span>
						)}
					</div>
				</div>

				<div
					aria-live="polite"
					className="min-h-0 w-full flex-1 overflow-auto overflow-x-hidden px-3 pt-3 pb-6 text-sm"
					ref={containerRef}
				>
					{displayEvents.length === 0 ? (
						<div className="text-muted-foreground">{t("common.noEvents")}</div>
					) : (
						<ul className="space-y-2">
							{displayEvents.map((event) => {
								const isVisible = visibleEvents.includes(event.id);
								const isReasoning = event.metadata?.isReasoning === "true";

								if (isReasoning) {
									const isExpanded = expandedReasoning.has(event.id);
									return (
										<li
											className={cn("transition-all duration-300", {
												"translate-y-0 opacity-100": isVisible,
												"translate-y-2 opacity-0": !isVisible,
											})}
											key={event.id}
										>
											<div className="overflow-hidden rounded-md border border-amber-500/30 bg-amber-500/5">
												<button
													className="flex w-full items-center justify-between px-3 py-2 text-left transition-colors hover:bg-amber-500/10"
													onClick={() => toggleReasoning(event.id)}
													type="button"
												>
													<div className="flex items-center gap-2">
														<span className="font-medium text-amber-700 text-sm">
															{t("activity.thoughtProcess", "Thought process")}
														</span>
														<span className="text-muted-foreground text-xs">
															{event.timestamp.toLocaleTimeString()}
														</span>
													</div>
													{isExpanded ? (
														<ChevronUp className="h-4 w-4 text-muted-foreground" />
													) : (
														<ChevronDown className="h-4 w-4 text-muted-foreground" />
													)}
												</button>
												{isExpanded && (
													<div
														className="max-h-48 overflow-y-auto border-amber-500/30 border-t bg-amber-500/5 px-4 py-2"
														data-reasoning-id={event.id}
													>
														{event.message ? (
															<MarkdownRenderer
																className="text-muted-foreground"
																content={event.message}
															/>
														) : (
															<p className="font-mono text-muted-foreground text-xs italic">
																{t(
																	"activity.reasoningPlaceholder",
																	"Waiting for reasoning…"
																)}
															</p>
														)}
													</div>
												)}
											</div>
										</li>
									);
								}

								// Check if this is a tool event
								const toolData = parseToolEvent(event);
								if (toolData && toolData.prefixIcon) {
									// Use repository-based icon (BookOpen for docs, Code for codebase)
									const DisplayIcon = toolData.prefixIcon;
									// Color based on prefix: amber for docs, blue for code
									const iconColor = event.metadata?.path?.startsWith("docs:")
										? "text-amber-600"
										: "text-blue-600";

									return (
										<li
											className={cn(
												"flex items-start gap-2 transition-all duration-300",
												{
													"translate-y-0 opacity-100": isVisible,
													"translate-y-2 opacity-0": !isVisible,
												}
											)}
											key={event.id}
										>
											<div className="mt-0.5 shrink-0">
												<DisplayIcon className={cn("h-4 w-4", iconColor)} />
											</div>
											<span className="min-w-0 flex-1 break-words text-foreground/90">
												<Trans
													components={[
														<code
															className="rounded bg-muted px-1 py-0.5 text-xs"
															key="path-code"
														/>,
													]}
													i18nKey={`tools.${toolData.toolType}`}
													values={{
														path: toolData.params.path,
														pattern: toolData.params.pattern,
													}}
												/>
											</span>
											<span className="ml-auto shrink-0 text-muted-foreground text-xs">
												{event.timestamp.toLocaleTimeString()}
											</span>
										</li>
									);
								}

								// Regular event display
								return (
									<li
										className={cn(
											"flex items-start gap-2 transition-all duration-300",
											{
												"translate-y-0 opacity-100": isVisible,
												"translate-y-2 opacity-0": !isVisible,
											}
										)}
										key={event.id}
									>
										<span
											className={cn(
												"inline-flex shrink-0 items-center rounded px-2 py-0.5 font-medium text-xs",
												{
													error: "bg-red-500/15 text-red-600",
													warn: "bg-yellow-500/15 text-yellow-700",
													info: "bg-blue-500/15 text-blue-700",
													success: "bg-emerald-500/15 text-emerald-700",
												}[event.type] || "bg-emerald-500/15 text-emerald-700"
											)}
										>
											{event.type}
										</span>
										<span className="min-w-0 flex-1 break-words text-foreground/90">
											{event.message}
										</span>
										<span className="ml-auto shrink-0 text-muted-foreground text-xs">
											{event.timestamp.toLocaleTimeString()}
										</span>
									</li>
								);
							})}
						</ul>
					)}

					{/* Loading indicator */}
					{inProgress && (
						<div className="mt-4 flex items-center justify-center py-4">
							<div className="flex space-x-1">
								<div className="h-2 w-2 animate-bounce rounded-full bg-primary [animation-delay:-0.3s]" />
								<div className="h-2 w-2 animate-bounce rounded-full bg-primary [animation-delay:-0.15s]" />
								<div className="h-2 w-2 animate-bounce rounded-full bg-primary" />
							</div>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}
