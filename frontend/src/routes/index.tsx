import type { models } from "@go/models";
import { List as listSessions } from "@go/services/generationSessionService";
import { List as listProjects } from "@go/services/repoLinkService";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Clock, Loader2, PlayCircle, PlusCircle } from "lucide-react";
import { useCallback, useEffect, useId, useMemo, useState } from "react";
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
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useDocGenerationStore } from "@/stores/docGeneration";

const PROJECT_FETCH_LIMIT = 100;
const PROJECT_FETCH_OFFSET = 0;

type PendingSessionSummary = {
	id: string;
	projectId: number;
	projectName: string;
	sourceBranch: string;
	targetBranch: string;
	updatedAt: string | null;
};

export const Route = createFileRoute("/")({
	component: Home,
});

function Home() {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const [projects, setProjects] = useState<models.RepoLink[]>([]);
	const [projectsLoading, setProjectsLoading] = useState(true);
	const [sessionsLoading, setSessionsLoading] = useState(false);
	const [pendingSessions, setPendingSessions] = useState<
		PendingSessionSummary[]
	>([]);
	const [restoringKey, setRestoringKey] = useState<string | null>(null);
	const [dialogOpen, setDialogOpen] = useState(false);
	const [selectedProjectId, setSelectedProjectId] = useState<string>("");
	const sessionMeta = useDocGenerationStore((state) => state.sessionMeta);
	const activeSessions = useDocGenerationStore((state) => state.activeSession);
	const restoreSession = useDocGenerationStore((state) => state.restoreSession);
	const setActiveSession = useDocGenerationStore(
		(state) => state.setActiveSession
	);

	const loadProjects = useCallback(() => {
		setProjectsLoading(true);
		Promise.resolve(listProjects(PROJECT_FETCH_LIMIT, PROJECT_FETCH_OFFSET))
			.then((res) => {
				if (!Array.isArray(res)) {
					setProjects([]);
					return;
				}
				setProjects(res);
			})
			.catch(() => {
				setProjects([]);
			})
			.finally(() => {
				setProjectsLoading(false);
			});
	}, []);

	useEffect(() => {
		loadProjects();
	}, [loadProjects]);

	useEffect(() => {
		if (projects.length > 0 && !selectedProjectId) {
			setSelectedProjectId(String(projects[0].ID));
		}
	}, [projects, selectedProjectId]);

	const loadPendingSessions = useCallback(() => {
		if (projects.length === 0) {
			setPendingSessions([]);
			return;
		}
		setSessionsLoading(true);
		Promise.all(
			projects.map((project) =>
				Promise.resolve(listSessions(Number(project.ID)))
					.then((sessions) => ({
						project,
						sessions: Array.isArray(sessions) ? sessions : [],
					}))
					.catch(() => ({ project, sessions: [] }))
			)
		)
			.then((results) => {
				const summaries: PendingSessionSummary[] = [];
				for (const result of results) {
					for (const session of result.sessions) {
						const key = `${result.project.ID}:${session.SourceBranch}:${session.TargetBranch}`;
						summaries.push({
							id: key,
							projectId: Number(result.project.ID),
							projectName: result.project.ProjectName,
							sourceBranch: session.SourceBranch,
							targetBranch: session.TargetBranch,
							updatedAt: session.UpdatedAt ? String(session.UpdatedAt) : null,
						});
					}
				}
				summaries.sort((a, b) => {
					const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
					const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
					return bTime - aTime;
				});
				setPendingSessions(summaries);
			})
			.finally(() => {
				setSessionsLoading(false);
			});
	}, [projects]);

	useEffect(() => {
		loadPendingSessions();
	}, [loadPendingSessions]);

	const runningSessions = useMemo(
		() =>
			Object.entries(sessionMeta)
				.filter(
					([, meta]) =>
						meta.status === "running" || meta.status === "committing"
				)
				.map(([sessionKey, meta]) => ({
					sessionKey,
					meta,
				})),
		[sessionMeta]
	);

	const formatUpdated = useCallback((raw: string | null) => {
		if (!raw) {
			return null;
		}
		const parsed = new Date(raw);
		if (Number.isNaN(parsed.getTime())) {
			return null;
		}
		return parsed.toLocaleString();
	}, []);

	const handleResumePending = useCallback(
		async (summary: PendingSessionSummary) => {
			const key = `pending:${summary.id}`;
			setRestoringKey(key);
			try {
				await restoreSession(
					summary.projectId,
					summary.sourceBranch,
					summary.targetBranch
				);
				navigate({
					to: "/projects/$projectId",
					params: { projectId: String(summary.projectId) },
				});
			} finally {
				setRestoringKey(null);
			}
		},
		[navigate, restoreSession]
	);

	const handleResumeRunning = useCallback(
		async (
			sessionKey: string,
			meta: { projectId: number; sourceBranch: string; targetBranch: string }
		) => {
			const key = `running:${sessionKey}`;
			setRestoringKey(key);
			setActiveSession(meta.projectId, sessionKey);
			try {
				await restoreSession(
					meta.projectId,
					meta.sourceBranch,
					meta.targetBranch
				);
				navigate({
					to: "/projects/$projectId",
					params: { projectId: String(meta.projectId) },
				});
			} finally {
				setRestoringKey(null);
			}
		},
		[navigate, restoreSession, setActiveSession]
	);

	const projectSelectId = useId();

	const handleOpenDialog = () => {
		if (projects.length === 0) {
			return;
		}
		if (!selectedProjectId && projects[0]) {
			setSelectedProjectId(String(projects[0].ID));
		}
		setDialogOpen(true);
	};

	const handleStartSession = () => {
		if (!selectedProjectId) {
			return;
		}
		setDialogOpen(false);
		navigate({
			to: "/projects/$projectId",
			params: { projectId: selectedProjectId },
		});
	};

	const runningVisible = runningSessions.length > 0;
	const pendingVisible = sessionsLoading || pendingSessions.length > 0;
	const projectsUnavailable = projectsLoading || projects.length === 0;

	return (
		<>
			<div className="flex h-full w-full justify-center overflow-y-auto bg-background">
				<div className="flex w-full max-w-6xl flex-col gap-6 p-6 md:p-10">
					<Card className="border border-border/60">
						<CardHeader className="border-border/60 border-b">
							<CardTitle className="text-2xl text-foreground">
								{t("common.appName")}
							</CardTitle>
							<CardDescription>{t("home.welcomeMessage")}</CardDescription>
						</CardHeader>
						<CardContent className="flex flex-col gap-4">
							<p className="text-muted-foreground text-sm">
								{t("home.newSessionDescription")}
							</p>
							<Button
								disabled={projectsUnavailable}
								onClick={handleOpenDialog}
								size="lg"
								type="button"
							>
								{projectsLoading ? (
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
								) : (
									<PlusCircle className="mr-2 h-5 w-5" />
								)}
								{projectsLoading
									? t("home.loadingProjects")
									: t("home.newSessionButton")}
							</Button>
						</CardContent>
					</Card>

					{runningVisible && (
						<Card className="border border-border/60">
							<CardHeader className="border-border/60 border-b">
								<CardTitle>{t("home.runningSessionsTitle")}</CardTitle>
								<CardDescription>
									{t("home.runningSessionsDescription")}
								</CardDescription>
							</CardHeader>
							<CardContent className="pt-6">
								<ul className="flex max-h-[500px] flex-col gap-4 overflow-y-auto">
									{runningSessions.map(({ sessionKey, meta }) => {
										const restoreKey = `running:${sessionKey}`;
										const isRestoring = restoringKey === restoreKey;
										const branchLabel = meta.targetBranch
											? `${meta.sourceBranch} -> ${meta.targetBranch}`
											: meta.sourceBranch;
										const statusLabel =
											meta.status === "committing"
												? t("generations.statusCommitting")
												: t("generations.statusRunning");
										const projectKey = String(meta.projectId);
										const isActive = activeSessions[projectKey] === sessionKey;
										return (
											<li
												className={`flex flex-col gap-3 rounded-xl border p-4 sm:flex-row sm:items-center sm:justify-between ${
													isActive
														? "border-primary/60 bg-primary/5"
														: "border-border/60 bg-card/60"
												}`}
												key={sessionKey}
											>
												<div className="space-y-1">
													<p className="font-medium text-sm">
														{meta.projectName}
													</p>
													<p className="text-muted-foreground text-xs">
														{branchLabel}
													</p>
													<p className="font-medium text-primary text-xs">
														{statusLabel}
													</p>
												</div>
												<div className="flex flex-col gap-2">
													<Button
														onClick={() =>
															navigate({
																to: "/projects/$projectId",
																params: { projectId: projectKey },
															})
														}
														size="sm"
														type="button"
														variant="outline"
													>
														{t("home.viewProject")}
													</Button>
													<Button
														disabled={isRestoring}
														onClick={() =>
															handleResumeRunning(sessionKey, {
																projectId: meta.projectId,
																sourceBranch: meta.sourceBranch,
																targetBranch: meta.targetBranch,
															})
														}
														size="sm"
														type="button"
													>
														{isRestoring ? (
															<Loader2 className="mr-2 h-4 w-4 animate-spin" />
														) : (
															<PlayCircle className="mr-2 h-4 w-4" />
														)}
														{t("home.resumeSession")}
													</Button>
												</div>
											</li>
										);
									})}
								</ul>
							</CardContent>
						</Card>
					)}

					{pendingVisible && (
						<Card className="border border-border/60">
							<CardHeader className="border-border/60 border-b">
								<CardTitle>{t("home.pendingSessionsTitle")}</CardTitle>
								<CardDescription>
									{t("home.pendingSessionsDescription")}
								</CardDescription>
							</CardHeader>
							<CardContent className="flex flex-col gap-4">
								{sessionsLoading && (
									<div className="flex items-center gap-2 text-muted-foreground text-sm">
										<Loader2 className="h-4 w-4 animate-spin" />
										{t("home.loadingSessions")}
									</div>
								)}
								<ul className="flex max-h-[500px] flex-col gap-4 overflow-y-auto">
									{pendingSessions.map((summary) => {
										const restoreKey = `pending:${summary.id}`;
										const isRestoring = restoringKey === restoreKey;
										const updatedLabel = formatUpdated(summary.updatedAt);
										const branchLabel = summary.targetBranch
											? `${summary.sourceBranch} -> ${summary.targetBranch}`
											: summary.sourceBranch;
										return (
											<li
												className="flex flex-col gap-3 rounded-xl border border-border/60 bg-card/60 p-4 lg:flex-row lg:items-center lg:justify-between"
												key={summary.id}
											>
												<div className="space-y-1">
													<p className="font-medium text-sm">
														{summary.projectName}
													</p>
													<p className="text-muted-foreground text-xs">
														{branchLabel}
													</p>
													{updatedLabel && (
														<span className="flex items-center gap-1 text-[11px] text-muted-foreground">
															<Clock className="h-3.5 w-3.5" />
															{updatedLabel}
														</span>
													)}
												</div>
												<div className="flex flex-col gap-2">
													<Button
														onClick={() =>
															navigate({
																to: "/projects/$projectId",
																params: {
																	projectId: String(summary.projectId),
																},
															})
														}
														size="sm"
														type="button"
														variant="outline"
													>
														{t("home.viewProject")}
													</Button>
													<Button
														disabled={isRestoring}
														onClick={() => handleResumePending(summary)}
														size="sm"
														type="button"
													>
														{isRestoring ? (
															<Loader2 className="mr-2 h-4 w-4 animate-spin" />
														) : (
															<PlayCircle className="mr-2 h-4 w-4" />
														)}
														{t("home.resumeSession")}
													</Button>
												</div>
											</li>
										);
									})}
								</ul>
							</CardContent>
						</Card>
					)}
				</div>
			</div>

			<Dialog onOpenChange={setDialogOpen} open={dialogOpen}>
				<DialogContent className="sm:max-w-md">
					<DialogHeader>
						<DialogTitle>{t("home.projectPickerTitle")}</DialogTitle>
						<DialogDescription>
							{t("home.projectPickerDescription")}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-3">
						<div className="space-y-1">
							<Label htmlFor={projectSelectId}>
								{t("home.projectSelectLabel")}
							</Label>
							<Select
								onValueChange={(value) => setSelectedProjectId(value)}
								value={selectedProjectId}
							>
								<SelectTrigger id={projectSelectId}>
									<SelectValue
										placeholder={t("home.projectSelectPlaceholder")}
									/>
								</SelectTrigger>
								<SelectContent>
									{projects.map((project) => (
										<SelectItem key={project.ID} value={String(project.ID)}>
											{project.ProjectName}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
					</div>
					<DialogFooter className="gap-2 pt-4">
						<Button
							onClick={() => setDialogOpen(false)}
							type="button"
							variant="outline"
						>
							{t("common.cancel")}
						</Button>
						<Button
							disabled={!selectedProjectId}
							onClick={handleStartSession}
							type="button"
						>
							{t("home.projectPickerConfirm")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}
