import { Clock, GitBranch } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { GetAvailableTabSessions } from "@go/services/ClientService";
import type { services } from "@go/models";

export type SessionSelectorModalProps = {
	projectId: number;
	open: boolean;
	onClose: () => void;
	onSelectSession: (sessionKey: string) => void;
};

export function SessionSelectorModal({
	projectId,
	open,
	onClose,
	onSelectSession,
}: SessionSelectorModalProps) {
	const { t } = useTranslation();
	const [sessions, setSessions] = useState<services.SessionInfo[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const loadSessions = useCallback(async () => {
		if (!open) {
			return;
		}
		setLoading(true);
		setError(null);
		try {
			const availableSessions = await GetAvailableTabSessions(projectId);
			setSessions(availableSessions);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to load sessions"
			);
		} finally {
			setLoading(false);
		}
	}, [projectId, open]);

	useEffect(() => {
		if (open) {
			loadSessions();
		}
	}, [open, loadSessions]);

	const handleSelectSession = (session: services.SessionInfo) => {
		const sessionKey = `${session.projectId}:${session.sourceBranch}`;
		onSelectSession(sessionKey);
		onClose();
	};

	const formatDate = (dateStr: string) => {
		try {
			const date = new Date(dateStr);
			return date.toLocaleString();
		} catch {
			return dateStr;
		}
	};

	return (
		<Dialog onOpenChange={onClose} open={open}>
			<DialogContent className="max-h-[80vh] w-auto max-w-2xl overflow-y-auto">
				<DialogHeader>
					<DialogTitle className="text-foreground text-lg">
						{t("sessionSelector.title")}
					</DialogTitle>
					<DialogDescription className="text-muted-foreground">
						{t("sessionSelector.description")}
					</DialogDescription>
				</DialogHeader>

				<div className="flex flex-col gap-4">
					{loading && (
						<div className="flex items-center justify-center p-8">
							<p className="text-muted-foreground">{t("generations.loading")}</p>
						</div>
					)}

					{error && (
						<div className="rounded-md border border-destructive bg-destructive/10 p-4">
							<p className="text-destructive text-sm">{error}</p>
						</div>
					)}

					{!loading && !error && sessions.length === 0 && (
						<div className="flex flex-col items-center justify-center gap-2 p-8">
							<p className="text-foreground font-medium">
								{t("sessionSelector.noSessions")}
							</p>
							<p className="text-muted-foreground text-sm">
								{t("sessionSelector.noSessionsDescription")}
							</p>
						</div>
					)}

					{!loading && !error && sessions.length > 0 && (
						<div className="flex flex-col gap-3">
							{sessions.map((session) => (
								<Card
									key={`${session.projectId}:${session.sourceBranch}`}
									className="cursor-pointer transition-colors hover:border-primary"
									onClick={() => handleSelectSession(session)}
								>
									<CardHeader>
										<div className="flex items-start justify-between gap-4">
											<div className="flex-1">
												<CardTitle className="flex items-center gap-2 text-base">
													<GitBranch className="h-4 w-4" />
													<span>{session.sourceBranch}</span>
													{session.isRunning && (
														<span className="rounded-full bg-primary px-2 py-0.5 font-medium text-primary-foreground text-xs">
															{t("generations.running")}
														</span>
													)}
												</CardTitle>
												<CardDescription className="mt-1 flex items-center gap-2">
													<Clock className="h-3 w-3" />
													<span>
														{t("generations.lastUpdated")}:{" "}
														{formatDate(session.updatedAt)}
													</span>
												</CardDescription>
											</div>
										</div>
									</CardHeader>
									<CardContent>
										<div className="flex flex-col gap-2 text-sm">
											{session.targetBranch && (
												<div className="flex items-center gap-2">
													<span className="text-muted-foreground">
														{t("common.targetBranch")}:
													</span>
													<span className="font-medium text-foreground">
														{session.targetBranch}
													</span>
												</div>
											)}
											<div className="flex items-center gap-2">
												<span className="text-muted-foreground">
													{t("common.docsBranch")}:
												</span>
												<span className="font-medium text-foreground">
													{session.docsBranch}
												</span>
											</div>
											<div className="flex items-center gap-2">
												<span className="text-muted-foreground">
													{t("common.llmModel")}:
												</span>
												<span className="font-medium text-foreground">
													{session.provider} / {session.modelKey}
												</span>
											</div>
										</div>
									</CardContent>
								</Card>
							))}
						</div>
					)}
				</div>

				<div className="flex justify-end gap-2 pt-4">
					<Button onClick={onClose} type="button" variant="outline">
						{t("common.cancel")}
					</Button>
				</div>
			</DialogContent>
		</Dialog>
	);
}
