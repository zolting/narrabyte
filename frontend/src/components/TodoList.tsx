import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { TodoItem } from "@/types/events";
import {
	CheckCircle2,
	Circle,
	Loader2,
	XCircle,
} from "lucide-react";

export function TodoList({ todos }: { todos: TodoItem[] }) {
	const { t } = useTranslation();
	const containerRef = useRef<HTMLDivElement | null>(null);
	const previousTodoCountRef = useRef(0);
	const [visibleTodos, setVisibleTodos] = useState<string[]>([]);

	// Animate new todos appearing
	useEffect(() => {
		const newTodos = todos.slice(previousTodoCountRef.current);
		previousTodoCountRef.current = todos.length;

		if (newTodos.length === 0) {
			return;
		}

		const timeouts = newTodos.map((todo, index) =>
			window.setTimeout(() => {
				setVisibleTodos((prev) => {
					const key = `${todo.content}-${todo.status}`;
					if (prev.includes(key)) {
						return prev;
					}
					return [...prev, key];
				});
			}, index * 100)
		);

		return () => {
			for (const timeout of timeouts) {
				window.clearTimeout(timeout);
			}
		};
	}, [todos]);

	// Reset visible todos when list changes
	useEffect(() => {
		setVisibleTodos(todos.map((t) => `${t.content}-${t.status}`));
	}, [todos]);

	// Calculate counts
	const pendingCount = todos.filter((t) => t.status === "pending").length;
	const inProgressCount = todos.filter((t) => t.status === "in_progress").length;
	const completedCount = todos.filter((t) => t.status === "completed").length;

	// Get icon for todo status
	const getStatusIcon = (status: TodoItem["status"]) => {
		switch (status) {
			case "completed":
				return <CheckCircle2 className="h-4 w-4 text-emerald-600" />;
			case "in_progress":
				return <Loader2 className="h-4 w-4 animate-spin text-blue-600" />;
			case "cancelled":
				return <XCircle className="h-4 w-4 text-muted-foreground" />;
			case "pending":
			default:
				return <Circle className="h-4 w-4 text-muted-foreground" />;
		}
	};

	if (todos.length === 0) {
		return null;
	}

	return (
		<div className="flex min-h-0 flex-col gap-2">
			<div className="flex items-center justify-between">
				<span className="font-medium text-foreground text-sm">
					{t("todos.title", "Task Progress")}
				</span>
				<div className="flex items-center gap-3 text-xs">
					{inProgressCount > 0 && (
						<span className="text-blue-600">
							{t("todos.inProgress", "{{count}} in progress", { count: inProgressCount })}
						</span>
					)}
					{pendingCount > 0 && (
						<span className="text-muted-foreground">
							{t("todos.pending", "{{count}} pending", { count: pendingCount })}
						</span>
					)}
					{completedCount > 0 && (
						<span className="text-emerald-600">
							{t("todos.completed", "{{count}} completed", { count: completedCount })}
						</span>
					)}
				</div>
			</div>
			<div className="overflow-hidden rounded-md border border-border bg-muted/30">
				<div
					aria-live="polite"
					className="max-h-64 w-full overflow-auto overflow-x-hidden px-3 pt-3 pb-3 text-sm"
					ref={containerRef}
				>
					<ul className="space-y-2">
						{todos.map((todo, index) => {
							const key = `${todo.content}-${todo.status}`;
							const isVisible = visibleTodos.includes(key);
							const displayText = todo.status === "in_progress" ? todo.activeForm : todo.content;

							return (
								<li
									className={cn(
										"flex items-start gap-2.5 rounded-md px-2 py-2 transition-all duration-300",
										{
											"translate-y-0 opacity-100": isVisible,
											"translate-y-2 opacity-0": !isVisible,
											"bg-emerald-500/10": todo.status === "completed",
											"bg-blue-500/10": todo.status === "in_progress",
											"bg-muted/50": todo.status === "pending",
											"bg-muted/30 opacity-60": todo.status === "cancelled",
										}
									)}
									key={`${key}-${index}`}
								>
									<div className="mt-0.5 shrink-0">
										{getStatusIcon(todo.status)}
									</div>
									<span
										className={cn("min-w-0 flex-1 break-words", {
											"text-foreground": todo.status !== "cancelled",
											"text-muted-foreground line-through": todo.status === "cancelled",
											"font-medium": todo.status === "in_progress",
										})}
									>
										{displayText}
									</span>
								</li>
							);
						})}
					</ul>
				</div>
			</div>
		</div>
	);
}
