import {useTranslation} from "react-i18next";
import {useDemoEventsStore} from "@/stores/demoEvents";
import {Button} from "./ui/button";

export default function DemoEvents() {
	const { t } = useTranslation();
	const events = useDemoEventsStore((s) => s.events);
	const isListening = useDemoEventsStore((s) => s.isListening);
	const startDemo = useDemoEventsStore((s) => s.start);
	const stopDemo = useDemoEventsStore((s) => s.stop);
	const clearEvents = useDemoEventsStore((s) => s.clear);

	return (
		<div className="mt-6 space-y-3 border-t pt-4">
			<div className="flex items-center justify-between">
				<div className="font-semibold">{t("demoEvents.title")}</div>
				<div className="flex gap-2">
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
			<div className="max-h-56 overflow-auto rounded-md border bg-muted/30 p-3 text-sm">
				{events.length === 0 ? (
					<div className="text-muted-foreground">
						{t("demoEvents.noEvents")}
					</div>
				) : (
					<ul className="space-y-1">
						{events.map((e) => (
							<li className="flex items-center gap-2" key={e.id}>
								<span
									className={`inline-flex items-center rounded px-2 py-0.5 font-medium text-xs ${
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
								<span className="text-foreground/90">{e.message}</span>
								<span className="ml-auto text-muted-foreground text-xs">
									{new Date(e.timestamp).toLocaleTimeString()}
								</span>
							</li>
						))}
					</ul>
				)}
			</div>
		</div>
	);
}
