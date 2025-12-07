import type { models, services } from "@go/models";
import { DeleteByID, List } from "@go/services/generationSessionService";
import { createFileRoute } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { DeleteSessionDialog } from "@/components/DeleteSessionDialog";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardAction,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { useDocGenerationStore } from "@/stores/docGeneration";

export const Route = createFileRoute("/projects/$projectId/generations")({
	component: RouteComponent,
});

function RouteComponent() {
	const { t } = useTranslation();
	const { projectId } = Route.useParams();
	const [sessions, setSessions] = useState<models.GenerationSession[] | null>(
		null
	);
	const [loading, setLoading] = useState<boolean>(true);
	const [restoringId, setRestoringId] = useState<number | null>(null);
	const [deletingId, setDeletingId] = useState<number | null>(null);
	const clearSessionMeta = useDocGenerationStore((s) => s.clearSessionMeta);
	const navigate = Route.useNavigate();

	useEffect(() => {
		let mounted = true;
		setLoading(true);
		Promise.resolve(List(Number(projectId)))
			.then((list) => {
				if (!mounted) {
					return;
				}
				setSessions(list);
			})
			.finally(() => {
				setLoading(false);
			});
		return () => {
			mounted = false;
		};
	}, [projectId]);

	const handleBack = useCallback(() => {
		navigate({ to: "/projects/$projectId", params: { projectId } });
	}, [navigate, projectId]);

	const handleResume = useCallback(
		async (s: models.GenerationSession) => {
			setRestoringId(Number(s.ID));
			try {
				// Construct SessionInfo for the restore event
				const sessionInfo: services.SessionInfo = {
					id: s.ID,
					sessionKey: `session:${s.ID}`,
					projectId: Number(projectId),
					sourceBranch: s.SourceBranch,
					targetBranch: s.TargetBranch,
					modelKey: s.ModelKey ?? "",
					provider: s.Provider ?? "",
					docsBranch: s.DocsBranch ?? "",
					inTab: false,
					isRunning: false,
					createdAt: "",
					updatedAt: "",
				};
				navigate({ to: "/projects/$projectId", params: { projectId } });
				window.setTimeout(() => {
					window.dispatchEvent(
						new CustomEvent("ui:restore-session-tab", {
							detail: { projectId: Number(projectId), sessionInfo },
						})
					);
				}, 100);
			} finally {
				setRestoringId(null);
			}
		},
		[navigate, projectId]
	);

	const formatUpdated = useCallback((raw: unknown) => {
		if (!raw) {
			return null;
		}
		try {
			const d = new Date(raw as string);
			if (Number.isNaN(d.getTime())) {
				return null;
			}
			return d.toLocaleString();
		} catch {
			return null;
		}
	}, []);

	const refreshSessions = useCallback(() => {
		setLoading(true);
		Promise.resolve(List(Number(projectId)))
			.then((list) => setSessions(list))
			.finally(() => setLoading(false));
	}, [projectId]);

	const handleDelete = useCallback(
		async (s: models.GenerationSession) => {
			setDeletingId(Number(s.ID));
			try {
				await DeleteByID(Number(s.ID));
				clearSessionMeta(Number(projectId), s.SourceBranch);
				await refreshSessions();
			} finally {
				setDeletingId(null);
			}
		},
		[projectId, refreshSessions, clearSessionMeta]
	);

	return (
		<div className="flex h-full flex-col gap-6 overflow-hidden p-4 pt-0">
			<section className="flex min-h-0 flex-1 flex-col gap-6 overflow-hidden rounded-lg border border-border bg-card p-4">
				<header className="sticky top-0 z-10 flex shrink-0 flex-col items-start gap-4 bg-card pb-2">
					<Button
						className="-ml-2 pl-0 text-muted-foreground hover:bg-transparent hover:text-foreground"
						onClick={handleBack}
						size="sm"
						type="button"
						variant="ghost"
					>
						<ArrowLeft className="mr-2 h-4 w-4" />
						{t("common.backToProject")}
					</Button>
					<div className="space-y-2">
						<h2 className="font-semibold text-foreground text-lg">
							{t("generations.title")}
						</h2>
						<p className="text-muted-foreground text-sm">
							{t("generations.description")}
						</p>
					</div>
				</header>

				<div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto overflow-x-hidden pr-2">
					{loading && (
						<div className="p-2 text-muted-foreground text-sm">
							{t("generations.loading")}
						</div>
					)}
					{!loading && (!sessions || sessions.length === 0) && (
						<div className="p-2 text-muted-foreground text-sm">
							{t("generations.noSessions")}
						</div>
					)}
					{!loading && sessions && sessions.length > 0 && (
						<div className="grid grid-cols-1 gap-3">
							{sessions.map((s) => (
								<Card key={String(s.ID)}>
									<CardHeader>
										<CardTitle>{t("generations.sessionTitle")}</CardTitle>
										<CardDescription>
											{t("generations.sessionDescription")}
										</CardDescription>
										<CardAction>
											<Button
												disabled={restoringId === Number(s.ID)}
												onClick={() => handleResume(s)}
												size="sm"
												type="button"
											>
												{t("generations.loadSession")}
											</Button>
										</CardAction>
									</CardHeader>
									<CardContent>
										<div className="text-sm">
											<div className="font-medium">
												{s.SourceBranch} → {s.TargetBranch}
											</div>
											<div className="text-muted-foreground text-xs">
												{t("common.branch")}:{" "}
												{s.DocsBranch || `docs/${s.SourceBranch}`}
											</div>
											<div className="text-muted-foreground text-xs">
												{t("generations.lastUpdated")}:{" "}
												{formatUpdated(s.UpdatedAt)}
											</div>
										</div>
									</CardContent>
									<div className="flex gap-2 px-6 pb-4">
										<DeleteSessionDialog
											isDeleting={deletingId === Number(s.ID)}
											onConfirm={() => handleDelete(s)}
										/>
									</div>
								</Card>
							))}
						</div>
					)}
				</div>
			</section>
		</div>
	);
}
