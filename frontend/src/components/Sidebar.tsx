import type { models } from "@go/models";
import { Init } from "@go/services/GitService";
import { LinkRepositories, List } from "@go/services/repoLinkService";
import { Link, useLocation } from "@tanstack/react-router";
import { Folder, Folders, Home, Plus, Settings } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import AddProjectDialog from "@/components/AddProjectDialog";
import { Button } from "@/components/ui/button";
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
	const [projects, setProjects] = useState<models.RepoLink[]>([]);
	const [loading, setLoading] = useState(false);
	const [isAddProjectOpen, setIsAddProjectOpen] = useState(false);

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

	const handleAddProject = async (data: {
		name: string;
		docDirectory: string;
		codebaseDirectory: string;
		initFumaDocs: boolean;
	}) => {
		try {
			await LinkRepositories(
				data.name,
				data.docDirectory,
				data.codebaseDirectory,
				data.initFumaDocs
			);

			toast(t("home.linkSuccess"));
			setIsAddProjectOpen(false);
			loadProjects();
		} catch (error) {
			const errorMsg = error instanceof Error ? error.message : String(error);
			if (errorMsg.startsWith("missing_git_repo")) {
				const missingDocRepo = errorMsg.endsWith("documentation");
				const dir = missingDocRepo
					? t("projectManager.docDirectory")
					: t("projectManager.codebaseDirectory");
				if (window.confirm(`${dir} + ${t("home.unexistantGitRepoCreate")}`)) {
					try {
						await Init(
							missingDocRepo ? data.docDirectory : data.codebaseDirectory
						);
						await LinkRepositories(
							data.name,
							data.docDirectory,
							data.codebaseDirectory,
							data.initFumaDocs
						);
						toast(t("home.linkSuccess"));
						setIsAddProjectOpen(false);
						loadProjects();
					} catch (initError) {
						console.error("Error initializing git repo:", initError);
						toast(t("home.initGitError"));
					}
					return;
				}
				return;
			}
			console.error("Error linking repositories:", error);
			toast(t("home.linkError"));
			return;
		}
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
											<SidebarMenuButton
												asChild
												isActive={
													location.pathname === `/projects/${projectId}`
												}
												size="sm"
												tooltip={p.ProjectName}
											>
												<Link params={{ projectId }} to="/projects/$projectId">
													<Folder size={14} />
													<span className="text-sm">{p.ProjectName}</span>
												</Link>
											</SidebarMenuButton>
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
