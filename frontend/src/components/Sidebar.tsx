import type { models } from "@go/models";
import { Init } from "@go/services/GitService";
import { Delete, LinkRepositories, List } from "@go/services/repoLinkService";
import { Link, useLocation, useNavigate } from "@tanstack/react-router";
import { Folder, Folders, Home, Plus, Settings, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import AddProjectDialog from "@/components/AddProjectDialog";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupAction,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
} from "@/components/ui/sidebar";

const MAX_REPOS = 100;
const REPO_OFFSET = 0;

function AppSidebarContent() {
	const { t } = useTranslation();
	const location = useLocation();
	const navigate = useNavigate();
	const [projects, setProjects] = useState<models.RepoLink[]>([]);
	const [loading, setLoading] = useState(false);
	const [isAddProjectOpen, setIsAddProjectOpen] = useState(false);
	const [projectToDelete, setProjectToDelete] =
		useState<models.RepoLink | null>(null);
	const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

	const loadProjects = useCallback(() => {
		setLoading(true);
		Promise.resolve(List(MAX_REPOS, REPO_OFFSET))
			.then((res) => {
				setProjects((res as models.RepoLink[]) ?? []);
			})
			.catch(() => {
				setProjects([]);
			})
			.finally(() => {
				setLoading(false);
			});
	}, []);

	useEffect(() => {
		loadProjects();
	}, [loadProjects]);

	// Helper function to validate project data
	const validateProjectData = (data: {
		name: string;
		docDirectory: string;
		codebaseDirectory: string;
	}) => {
		if (!(data.docDirectory && data.codebaseDirectory)) {
			toast(t("home.selectBothDirectories"));
			return false;
		}
		if (!data.name) {
			toast(t("home.projectNameRequired"));
			return false;
		}
		return true;
	};

	// Helper function to handle successful project linking
	const handleSuccess = () => {
		toast(t("home.linkSuccess"));
		setIsAddProjectOpen(false);
		loadProjects();
	};

	// Helper function to handle missing git repository error
	const handleMissingGitRepo = async (
		error: unknown,
		data: {
			name: string;
			docDirectory: string;
			codebaseDirectory: string;
			initFumaDocs: boolean;
			llmInstructions?: string;
		}
	) => {
		const errorMsg = error instanceof Error ? error.message : String(error);

		if (!errorMsg.startsWith("missing_git_repo")) {
			throw error;
		}

		const missingDocRepo = errorMsg.endsWith("documentation");
		const dir = missingDocRepo
			? t("projectManager.docDirectory")
			: t("projectManager.codebaseDirectory");

		const shouldCreate = window.confirm(
			`${dir} + ${t("home.unexistantGitRepoCreate")}`
		);
		if (!shouldCreate) {
			return false;
		}

		try {
			await Init(missingDocRepo ? data.docDirectory : data.codebaseDirectory);
			await LinkRepositories(
				data.name,
				data.docDirectory,
				data.codebaseDirectory,
				data.initFumaDocs,
				data.llmInstructions ?? ""
			);
			return true;
		} catch (initError) {
			console.error("Error initializing git repo:", initError);
			toast(t("home.initGitError"));
			return false;
		}
	};

	// Helper function to handle general errors
	const handleError = (error: unknown) => {
		console.error("Error linking repositories:", error);
		toast(t("home.linkError"));
	};

	const handleAddProject = async (data: {
		name: string;
		docDirectory: string;
		codebaseDirectory: string;
		initFumaDocs: boolean;
		llmInstructions?: string;
	}) => {
		if (!validateProjectData(data)) {
			return;
		}

		try {
			await LinkRepositories(
				data.name,
				data.docDirectory,
				data.codebaseDirectory,
				data.initFumaDocs,
				data.llmInstructions ?? ""
			);
			handleSuccess();
		} catch (error) {
			const success = await handleMissingGitRepo(error, data);
			if (success) {
				handleSuccess();
			} else {
				handleError(error);
			}
		}
	};

	const handleDeleteProject = async () => {
		if (!projectToDelete) {
			return;
		}

		try {
			await Delete(projectToDelete.ID);
			toast(t("sidebar.deleteSuccess"));
			loadProjects();

			// Navigate to home if we're currently viewing the deleted project
			if (location.pathname === `/projects/${projectToDelete.ID}`) {
				navigate({ to: "/" });
			}
		} catch (error) {
			console.error("Error deleting project:", error);
			toast(t("sidebar.deleteError"));
		} finally {
			setIsDeleteDialogOpen(false);
			setProjectToDelete(null);
		}
	};

	const openDeleteDialog = (project: models.RepoLink) => {
		setProjectToDelete(project);
		setIsDeleteDialogOpen(true);
	};

	return (
		<>
			<SidebarHeader className="border-sidebar-border border-b bg-sidebar-accent/20">
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton
							asChild
							className="h-12 justify-start font-semibold group-data-[collapsible=icon]:justify-center"
							isActive={location.pathname === "/"}
							size="lg"
							tooltip={t("sidebar.home")}
						>
							<Link to="/">
								<Home size={20} />
								<span className="font-semibold group-data-[collapsible=icon]:hidden">
									{t("sidebar.home")}
								</span>
							</Link>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarHeader>

			<SidebarContent className="pt-4">
				<SidebarGroup>
					<SidebarGroupLabel className="mb-1 px-1 font-semibold text-sidebar-foreground">
						<Folders size={18} />
						<span className="ml-1">{t("sidebar.projects")}</span>
					</SidebarGroupLabel>
					<SidebarGroupAction asChild>
						<Button
							aria-label={t("home.addProject")}
							className="h-5 w-5 p-0 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
							onClick={() => setIsAddProjectOpen(true)}
							size="sm"
							variant="ghost"
						>
							<Plus className="h-3.5 w-3.5" />
						</Button>
					</SidebarGroupAction>
					<SidebarGroupContent>
						<SidebarMenu>
							{loading && (
								<SidebarMenuItem>
									<div className="px-2 py-1 text-muted-foreground text-xs group-data-[collapsible=icon]:hidden">
										{t("sidebar.loading")}
									</div>
								</SidebarMenuItem>
							)}
							{!loading && projects.length === 0 && (
								<SidebarMenuItem>
									<div className="px-2 py-1 text-muted-foreground text-xs group-data-[collapsible=icon]:hidden">
										{t("sidebar.noProjects")}
									</div>
								</SidebarMenuItem>
							)}
							{!loading &&
								projects.map((p) => {
									const projectId = String(p.ID);
									return (
										<SidebarMenuItem key={`${projectId}-${p.ProjectName}`}>
											<ContextMenu>
												<ContextMenuTrigger>
													<SidebarMenuButton
														asChild
														isActive={
															location.pathname === `/projects/${projectId}`
														}
														size="sm"
														tooltip={p.ProjectName}
													>
														<Link
															params={{ projectId }}
															to="/projects/$projectId"
														>
															<Folder size={14} />
															<span className="text-sm">{p.ProjectName}</span>
														</Link>
													</SidebarMenuButton>
												</ContextMenuTrigger>
												<ContextMenuContent>
													<ContextMenuItem
														onSelect={() =>
															navigate({
																to: "/projects/$projectId/settings",
																params: { projectId },
															})
														}
													>
														<Settings size={14} />
														<span>{t("sidebar.projectSettings")}</span>
													</ContextMenuItem>
													<ContextMenuItem
														onSelect={() => openDeleteDialog(p)}
														variant="destructive"
													>
														<Trash2 size={14} />
														<span>{t("sidebar.deleteProject")}</span>
													</ContextMenuItem>
												</ContextMenuContent>
											</ContextMenu>
										</SidebarMenuItem>
									);
								})}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>

			<SidebarFooter className="border-sidebar-border border-t bg-sidebar-accent/10">
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton
							asChild
							className="hover:bg-sidebar-accent/30"
							isActive={location.pathname === "/settings"}
							size="sm"
							tooltip={t("sidebar.settings")}
						>
							<Link to="/settings">
								<Settings size={16} />
								<span className="text-sm">{t("sidebar.settings")}</span>
							</Link>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarFooter>

			<AddProjectDialog
				onClose={() => setIsAddProjectOpen(false)}
				onSubmit={handleAddProject}
				open={isAddProjectOpen}
			/>

			<AlertDialog
				onOpenChange={setIsDeleteDialogOpen}
				open={isDeleteDialogOpen}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{t("sidebar.deleteProjectTitle")}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{t("sidebar.deleteProjectDescription", {
								projectName: projectToDelete?.ProjectName,
							})}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>{t("sidebar.cancel")}</AlertDialogCancel>
						<AlertDialogAction
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
							onClick={handleDeleteProject}
						>
							{t("sidebar.delete")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}

export function AppSidebar() {
	return (
		<Sidebar collapsible="icon">
			<AppSidebarContent />
		</Sidebar>
	);
}
