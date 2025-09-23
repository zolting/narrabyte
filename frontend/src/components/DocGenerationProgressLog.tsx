import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { DocGenerationStatus } from "@/stores/docGeneration";
import type { DemoEvent } from "@/types/events";

export function DocGenerationProgressLog({
	events,
	status,
}: {
	events: DemoEvent[];
	status: DocGenerationStatus;
}) {
	const { t } = useTranslation();
	const containerRef = useRef<HTMLDivElement | null>(null);
	const previousEventCountRef = useRef(0);
	const [visibleEvents, setVisibleEvents] = useState<string[]>([]);

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

	useEffect(() => {
		setVisibleEvents(events.map((e) => e.id));
	}, [events]);

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

	const isRunning = status === "running";
	const isCommitting = status === "committing";
	const inProgress = isRunning || isCommitting;

	return (
		<div className="flex min-h-0 flex-1 flex-col gap-2">
			{inProgress && (
				<div className="flex items-center justify-between">
					<span className="font-medium text-foreground text-sm">
						{isRunning
							? t("common.generatingDocs", "Generating documentation…")
							: t("common.committingDocs", "Committing documentation…")}
					</span>
					<span className="text-muted-foreground text-xs">
						{t("common.inProgress", "In progress")}
					</span>
				</div>
			)}
			<div className="min-h-0 flex-1 overflow-hidden rounded-md border border-border bg-muted/30">
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
										className={`flex items-start gap-2 transition-all duration-300 ${
											isVisible
												? "translate-y-0 opacity-100"
												: "translate-y-2 opacity-0"
										}`}
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
