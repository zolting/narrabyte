import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { DemoEvent } from "@/types/events";

export function DocGenerationProgressLog({
	events,
	isRunning,
}: {
	events: DemoEvent[];
	isRunning: boolean;
}) {
	const { t } = useTranslation();
	const containerRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		const el = containerRef.current;
		if (!el) {
			return;
		}
		el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
	}, [events]);

	return (
		<div className="space-y-2">
			<div className="flex items-center justify-between">
				<span className="font-medium text-foreground text-sm">
					{isRunning
						? t("common.generatingDocs", "Generating documentation…")
						: t("common.recentActivity", "Recent activity")}
				</span>
				{isRunning && (
					<span className="text-muted-foreground text-xs">
						{t("common.inProgress", "In progress")}
					</span>
				)}
			</div>
			<div
				className="max-h-48 overflow-y-auto rounded-md border border-border bg-muted/40 p-3 text-xs"
				ref={containerRef}
			>
				{events.length === 0 ? (
					<div className="text-muted-foreground">
						{t("common.noEvents", "No tool activity yet.")}
					</div>
				) : (
					<ol className="space-y-1">
						{events.map((event) => (
							<li className="flex items-start gap-2" key={event.id}>
								<span
									className={cn(
										"rounded px-1.5 py-0.5 font-medium uppercase tracking-wide",
										"text-[10px]",
										{
											error: "bg-red-500/10 text-red-600",
											warn: "bg-yellow-500/15 text-yellow-700",
											debug: "bg-blue-500/15 text-blue-700",
											info: "bg-emerald-500/15 text-emerald-700",
										}[event.type] ?? "bg-muted text-foreground/80"
									)}
								>
									{event.type}
								</span>
								<span className="flex-1 text-foreground/90">
									{event.message}
								</span>
								<span className="text-[10px] text-muted-foreground">
									{event.timestamp.toLocaleTimeString()}
								</span>
							</li>
						))}
					</ol>
				)}
			</div>
		</div>
	);
}
