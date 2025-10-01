import { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useDocGenerationStore } from "@/stores/docGeneration";

export function DocRefinementChat({
	projectId,
	branch,
	hideHeader = false,
	className,
	style,
}: {
	projectId: number;
	branch: string;
	hideHeader?: boolean;
	className?: string;
	style?: React.CSSProperties;
}) {
	const messages = useDocGenerationStore((s) => s.messages);
	const chatOpen = useDocGenerationStore((s) => s.chatOpen);
	const refine = useDocGenerationStore((s) => s.refine);
	const status = useDocGenerationStore((s) => s.status);

	const [input, setInput] = useState("");
	const disabled = status === "running" || !branch || !projectId;

	const containerRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		if (!chatOpen) {
			return;
		}
		const el = containerRef.current;
		if (el) {
			el.scrollTop = el.scrollHeight;
		}
	}, [chatOpen, messages.length]);

	const pending = useMemo(
		() => messages.some((m) => m.status === "pending"),
		[messages]
	);

	const handleSend = async () => {
		const text = input.trim();
		if (!text) {
			return;
		}
		setInput("");
		await refine({ projectId, branch, instruction: text });
	};

	return (
		<section
			className={cn(
				"flex h-full flex-col rounded-lg border border-border",
				className
			)}
			style={style}
		>
			{!hideHeader && (
				<header className="flex items-center justify-between gap-2 border-border border-b px-3 py-2">
					<div className="font-medium text-sm">Chat</div>
				</header>
			)}

			{chatOpen && (
				<div className="flex min-h-0 flex-1 flex-col gap-2 p-3">
					<div
						className="min-h-0 flex-1 overflow-y-auto rounded-md border border-border bg-muted/30 p-2"
						ref={containerRef}
					>
						{messages.length === 0 ? (
							<div className="text-muted-foreground text-xs">
								Ask for refinements, e.g. "Make the settings.mdx persistence
								section more concise and add a concrete code example."
							</div>
						) : (
							<ul className="space-y-2">
								{messages.map((m) => (
									<li
										className={cn(
											"text-sm",
											m.role === "user"
												? "text-foreground"
												: "text-foreground/90"
										)}
										key={m.id}
									>
										<div
											className={cn(
												"rounded-md px-2 py-1",
												m.role === "user" ? "bg-accent/30" : "bg-muted/50"
											)}
										>
											<div className="whitespace-pre-wrap">{m.content}</div>
											{m.status === "pending" && (
												<div className="text-[10px] text-muted-foreground">
													sending…
												</div>
											)}
											{m.status === "error" && (
												<div className="text-[10px] text-destructive">
													failed
												</div>
											)}
										</div>
									</li>
								))}
							</ul>
						)}
					</div>

					<div className="flex items-end gap-2">
						<textarea
							className="max-h-40 min-h-[3rem] flex-1 resize-none rounded-md border border-border bg-background px-3 py-2 text-sm outline-none disabled:opacity-60"
							disabled={disabled}
							onChange={(e) => setInput(e.target.value)}
							onKeyDown={(e) => {
								if (e.key === "Enter" && !e.shiftKey) {
									e.preventDefault();
									handleSend();
								}
							}}
							placeholder="Describe the change you want…"
							rows={3}
							value={input}
						/>
						<Button
							className="font-semibold"
							disabled={disabled || !input.trim() || pending}
							onClick={handleSend}
							size="sm"
						>
							Send
						</Button>
					</div>
				</div>
			)}
		</section>
	);
}
