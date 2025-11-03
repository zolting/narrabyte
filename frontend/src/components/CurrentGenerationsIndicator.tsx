import { useNavigate } from "@tanstack/react-router";
import { Loader2, PlayCircle } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useDocGenerationStore } from "@/stores/docGeneration";

export function CurrentGenerationsIndicator() {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const [open, setOpen] = useState(false);
	const sessionMeta = useDocGenerationStore((state) => state.sessionMeta);
	const activeSessions = useDocGenerationStore((state) => state.activeSession);
	const restoreSession = useDocGenerationStore((state) => state.restoreSession);
	const setActiveSession = useDocGenerationStore((state) => state.setActiveSession);

	const runningSessions = useMemo(
		() =>
			Object.entries(sessionMeta).filter(([, meta]) =>
				meta.status === "running" || meta.status === "committing"
			),
		[sessionMeta]
	);

	if (runningSessions.length === 0) {
		return null;
	}

	const handleSelect = async (
		sessionKey: string,
		meta: {
			projectId: number;
			projectName: string;
			sourceBranch: string;
			targetBranch: string;
		}
	) => {
		setOpen(false);
		setActiveSession(meta.projectId, sessionKey);
		try {
			await restoreSession(
				meta.projectId,
				meta.sourceBranch,
				meta.targetBranch
			);
		} catch (error) {
			console.error("Failed to restore session", error);
		}
		navigate({
			to: "/projects/$projectId",
			params: { projectId: String(meta.projectId) },
		});
	};

	return (
		<Popover open={open} onOpenChange={setOpen}>
			<PopoverTrigger asChild>
				<Button variant="outline" size="sm">
					<Loader2 className="mr-2 h-4 w-4 animate-spin" />
					{t("generations.running", "Running")}
					<span className="ml-2 inline-flex h-5 min-w-[1.25rem] items-center justify-center rounded-full bg-primary/10 px-2 text-xs text-primary">
						{runningSessions.length}
					</span>
				</Button>
			</PopoverTrigger>
			<PopoverContent align="end" className="w-64 p-0" sideOffset={8}>
				<div className="border-b border-border px-3 py-2 text-sm font-medium">
					{t("generations.current", "Current generations")}
				</div>
				<ul className="max-h-60 divide-y divide-border overflow-auto">
					{runningSessions.map(([sessionKey, meta]) => {
						const projectKey = String(meta.projectId);
						const isActive = activeSessions[projectKey] === sessionKey;
						const statusLabel =
							meta.status === "committing"
								? t("generations.statusCommitting", "Committing")
								: t("generations.statusRunning", "Running");
						return (
							<li key={sessionKey}>
								<button
									className={`flex w-full items-start gap-2 px-3 py-2 text-left text-sm hover:bg-muted ${
										isActive ? "bg-muted" : ""
									}`}
									onClick={() => handleSelect(sessionKey, meta)}
									type="button"
								>
									<PlayCircle className="mt-0.5 h-4 w-4 text-primary" />
									<div className="flex flex-col">
										<span className="font-medium">{meta.projectName}</span>
										<span className="text-xs text-muted-foreground">
											{meta.sourceBranch}
											{meta.targetBranch
												? ` → ${meta.targetBranch}`
												: ""}
										</span>
										<span className="text-[10px] text-muted-foreground">
											{statusLabel}
										</span>
									</div>
								</button>
							</li>
						);
					})}
				</ul>
			</PopoverContent>
		</Popover>
	);
}
