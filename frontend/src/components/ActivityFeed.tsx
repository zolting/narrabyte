import {
	CheckCircle2,
	ChevronDown,
	ChevronUp,
	Circle,
	Loader2,
	XCircle,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { DocGenerationStatus } from "@/stores/docGeneration";
import type { DemoEvent, TodoItem } from "@/types/events";

export function ActivityFeed({
	events,
	todos,
	status,
}: {
	events: DemoEvent[];
	todos: TodoItem[];
	status: DocGenerationStatus;
}) {
	const { t } = useTranslation();
	const containerRef = useRef<HTMLDivElement | null>(null);
	const previousEventCountRef = useRef(0);
	const [visibleEvents, setVisibleEvents] = useState<string[]>([]);
	const [showAllTodos, setShowAllTodos] = useState(false);

	// Animate new events appearing
	useEffect(() => {
		const newEvents = events.slice(previousEventCountRef.current);
		previousEventCountRef.current = events.length;

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
	}, [events]);

	// Reset visible events when list changes
	useEffect(() => {
		setVisibleEvents(events.map((e) => e.id));
	}, [events]);

	// Auto-scroll to bottom when new events arrive
	useEffect(() => {
		if (events.length === 0) {
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
	}, [events.length]);

	// Find active todo
	const activeTodo = todos.find((todo) => todo.status === "in_progress");

	// Calculate todo counts
	const pendingCount = todos.filter((todo) => todo.status === "pending").length;
	const inProgressCount = todos.filter(
		(todo) => todo.status === "in_progress"
	).length;
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
								{t("todos.allCompleted", "All tasks completed")}
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
									{t("activity.allTasks", "All Tasks")}
								</span>
								<div className="flex items-center gap-2 text-xs">
									{pendingCount > 0 && (
										<span className="text-muted-foreground">
											{t("todos.pending", "{{count}} pending", {
												count: pendingCount,
											})}
										</span>
									)}
									{completedCount > 0 && (
										<span className="text-emerald-600">
											{t("todos.completed", "{{count}} completed", {
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
			<div className="min-h-0 flex-1 overflow-hidden rounded-md border border-border bg-muted/30">
				<div className="border-border border-b bg-muted/50 px-3 py-2">
					<div className="flex items-center justify-between">
						<span className="font-medium text-foreground text-sm">
							{inProgress
								? isRunning
									? t("common.generatingDocs", "Generating documentation…")
									: t("common.committingDocs", "Committing documentation…")
								: t("activity.toolActivity", "Tool Activity")}
						</span>
						{inProgress && (
							<span className="text-muted-foreground text-xs">
								{t("common.inProgress", "In progress")}
							</span>
						)}
					</div>
				</div>

				<div
					aria-live="polite"
					className="h-full w-full overflow-auto overflow-x-hidden px-3 pt-3 pb-6 text-sm"
					ref={containerRef}
				>
					{events.length === 0 ? (
						<div className="text-muted-foreground">
							{t("common.noEvents", "No tool activity yet.")}
						</div>
					) : (
						<ul className="space-y-1">
							{events.map((event) => {
								const isVisible = visibleEvents.includes(event.id);
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
													debug: "bg-blue-500/15 text-blue-700",
													info: "bg-emerald-500/15 text-emerald-700",
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
