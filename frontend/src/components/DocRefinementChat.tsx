import { Send } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { useDocGenerationStore } from "@/stores/docGeneration";
import { MarkdownRenderer } from "./MarkdownRenderer";

// Helper to strip internal instruction tags from message content for display
function cleanMessageContent(content: string): string {
	return content
		.replace(/<USER_INSTRUCTIONS>([\s\S]*?)<\/USER_INSTRUCTIONS>/g, "$1")
		.trim();
}

export function DocRefinementChat({
	sessionKey,
	hideHeader = false,
	className,
	style,
}: {
	sessionKey: string | null;
	hideHeader?: boolean;
	className?: string;
	style?: React.CSSProperties;
}) {
	const { t } = useTranslation();
	const messages = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.messages : null) ?? []
	);
	const chatOpen = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.chatOpen : false) ?? false
	);
	const refineDocs = useDocGenerationStore((s) => s.refine);
	const status = useDocGenerationStore(
		(s) => (sessionKey ? s.docStates[sessionKey]?.status : "idle") ?? "idle"
	);

	const [input, setInput] = useState("");
	const disabled = status === "running" || !sessionKey;

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
		if (!(text && sessionKey)) {
			return;
		}
		setInput("");
		await refineDocs({
			sessionKey,
			instruction: text,
		});
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
				<header className="flex items-center justify-between gap-2 border-border border-b px-2 py-1.5">
					<div className="font-medium text-xs">Chat</div>
				</header>
			)}

			{chatOpen && (
				<div className="flex min-h-0 flex-1 flex-col gap-2 p-2">
					<div
						className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden rounded-md border border-border bg-muted/30 p-2"
						ref={containerRef}
					>
						{messages.length === 0 ? (
							<div className="text-[11px] text-muted-foreground">
								{t("docRefinementChat.emptyState")}
							</div>
						) : (
							<ul className="space-y-4">
								{messages.map((m) => (
									<li
										className={cn(
											"flex w-full flex-col gap-1.5",
											m.role === "user" ? "items-end" : "items-start"
										)}
										key={m.id}
									>
										<div
											className={cn(
												"max-w-[90%] rounded-2xl px-3 py-2 text-xs",
												m.role === "user"
													? "rounded-br-sm bg-primary text-primary-foreground"
													: "rounded-bl-sm bg-secondary text-secondary-foreground"
											)}
										>
											<MarkdownRenderer
												content={cleanMessageContent(m.content)}
											/>
											{m.status === "pending" && (
												<div className="mt-1 text-[10px] opacity-70">
													sending…
												</div>
											)}
											{m.status === "error" && (
												<div className="mt-1 font-bold text-[10px] text-destructive">
													failed
												</div>
											)}
										</div>
									</li>
								))}
							</ul>
						)}
					</div>

					<div className="relative flex overflow-hidden border-2 border-border">
						<Textarea
							className="flex-1 resize-none border-0 px-3 py-2 text-xs shadow-none outline-none focus-visible:ring-0"
							disabled={disabled}
							onChange={(e) => setInput(e.target.value)}
							onKeyDown={(e) => {
								if (e.key === "Enter" && !e.shiftKey) {
									e.preventDefault();
									handleSend();
								}
							}}
							placeholder={t("docRefinementChat.placeholder")}
							rows={2}
							value={input}
						/>
						<div className="w-px bg-border" />
						<Button
							aria-label={t("common.submit")}
							className="h-full w-[3rem] shrink-0 rounded-none border-0 p-0"
							disabled={disabled || !input.trim() || pending}
							onClick={handleSend}
							size="sm"
						>
							<Send className="h-4 w-4" />
						</Button>
					</div>
				</div>
			)}
		</section>
	);
}
