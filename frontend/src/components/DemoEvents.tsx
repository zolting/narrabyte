import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useDemoEventsStore } from "@/stores/demoEvents";
import { Button } from "./ui/button";

export default function DemoEvents() {
	const { t } = useTranslation();
	const events = useDemoEventsStore((s) => s.events);
	const isListening = useDemoEventsStore((s) => s.isListening);
	const startDemo = useDemoEventsStore((s) => s.start);
	const stopDemo = useDemoEventsStore((s) => s.stop);
	const clearEvents = useDemoEventsStore((s) => s.clear);
	const eventsContainerRef = useRef<HTMLDivElement | null>(null);
	const previousEventCountRef = useRef(0);

	useEffect(() => {
		const container = eventsContainerRef.current;
		const hasNewEvent = events.length > previousEventCountRef.current;
		previousEventCountRef.current = events.length;

		if (!(container && hasNewEvent)) {
			return;
		}

		window.requestAnimationFrame(() => {
			container.scrollTo({ top: container.scrollHeight, behavior: "smooth" });
		});
	}, [events]);

	return (
		<div className="flex min-h-0 min-w-0 flex-1 flex-col gap-4 border-t pt-4">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div className="font-semibold">{t("demoEvents.title")}</div>
				<div className="flex flex-wrap gap-2">
					<Button
						disabled={isListening}
						onClick={startDemo}
						size="sm"
						variant="outline"
					>
						{isListening
							? t("demoEvents.streaming")
							: t("demoEvents.startDemo")}
					</Button>
					<Button
						disabled={!isListening}
						onClick={stopDemo}
						size="sm"
						variant="outline"
					>
						{t("demoEvents.stop")}
					</Button>
					<Button onClick={clearEvents} size="sm" variant="outline">
						{t("demoEvents.clear")}
					</Button>
				</div>
			</div>
			<div className="flex-1 min-h-0 overflow-hidden rounded-md border border-border bg-muted/30">
				<div
					aria-live="polite"
					className="h-full w-full overflow-auto overflow-x-hidden p-3 text-sm"
					ref={eventsContainerRef}
				>
					{events.length === 0 ? (
						<div className="text-muted-foreground">
							{t("demoEvents.noEvents")}
						</div>
					) : (
						<ul className="space-y-1">
							{events.map((e) => (
								<li className="flex items-start gap-2" key={e.id}>
									<span
										className={`inline-flex shrink-0 items-center rounded px-2 py-0.5 font-medium text-xs ${
											{
												error: "bg-red-500/15 text-red-600",
												warn: "bg-yellow-500/15 text-yellow-700",
												debug: "bg-blue-500/15 text-blue-700",
												info: "bg-emerald-500/15 text-emerald-700",
											}[e.type] || "bg-emerald-500/15 text-emerald-700"
										}`}
									>
										{e.type}
									</span>
									<span className="flex-1 min-w-0 break-words text-foreground/90">
										{e.message}
									</span>
									<span className="ml-auto shrink-0 text-muted-foreground text-xs">
										{e.timestamp.toLocaleTimeString()}
									</span>
								</li>
							))}
						</ul>
					)}
					{isListening && (
						<div className="flex justify-center items-center py-4 mt-4">
							<div className="flex space-x-1">
								<div className="w-2 h-2 bg-primary rounded-full animate-bounce [animation-delay:-0.3s]"></div>
								<div className="w-2 h-2 bg-primary rounded-full animate-bounce [animation-delay:-0.15s]"></div>
								<div className="w-2 h-2 bg-primary rounded-full animate-bounce"></div>
							</div>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}
