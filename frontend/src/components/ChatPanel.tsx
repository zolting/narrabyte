import { type FormEvent, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type Message = { id: string; role: "user" | "assistant"; content: string };

export function ChatPanel({ className }: { className?: string }) {
	const [messages, setMessages] = useState<Message[]>([]);
	const [input, setInput] = useState("");
	const listRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		// Auto-scroll to bottom on new messages
		const el = listRef.current;
		if (el) el.scrollTop = el.scrollHeight;
	}, [messages.length]);

	function onSubmit(e: FormEvent) {
		e.preventDefault();
		const trimmed = input.trim();
		if (!trimmed) return;
		setMessages((prev) => [
			...prev,
			{ id: crypto.randomUUID(), role: "user", content: trimmed },
		]);
		setInput("");
	}

	return (
		<section
			className={cn(
				"flex min-h-0 flex-col rounded-md border border-border bg-card",
				className
			)}
		>
			<header className="flex items-center justify-between border-border border-b p-3">
				<h3 className="font-medium text-foreground text-sm">Chat</h3>
			</header>

			<div
				className="min-h-0 flex-1 space-y-2 overflow-y-auto p-3"
				ref={listRef}
			>
				{messages.length === 0 ? (
					<div className="text-muted-foreground text-sm">
						Start typing to chat…
					</div>
				) : (
					messages.map((m) => (
						<div
							className={cn(
								"max-w-[85%] rounded-md px-3 py-2 text-sm",
								m.role === "user"
									? "ml-auto bg-primary text-primary-foreground"
									: "mr-auto bg-muted text-foreground"
							)}
							key={m.id}
						>
							{m.content}
						</div>
					))
				)}
			</div>

			<form
				className="flex items-center gap-2 border-border border-t p-3"
				onSubmit={onSubmit}
			>
				<input
					className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm outline-none ring-0 focus:border-ring/50"
					onChange={(e) => setInput(e.target.value)}
					placeholder="Type a message…"
					value={input}
				/>
				<Button
					className="text-foreground"
					size="sm"
					type="submit"
					variant="outline"
				>
					Send
				</Button>
			</form>
		</section>
	);
}
