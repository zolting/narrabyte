import type { models } from "@go/models";
import { Init } from "@go/services/GitService";
import { LinkRepositories, List } from "@go/services/repoLinkService";
import { Link } from "@tanstack/react-router";
import {
	ChevronLeft,
	ChevronRight,
	Plus,
	Home,
	FolderKanban,
	Settings
} from "lucide-react";
import { useCallback, useEffect, useState, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import AddProjectDialog from "@/components/AddProjectDialog";
import { Button } from "@/components/ui/button";

function NavItem(props: {
	to?: string;
	label: string;
	icon: ReactNode;
	collapsed: boolean;
	onClick?: () => void;
}) {
	const common =
		"rounded text-foreground/80 hover:bg-accent hover:text-foreground";
	const expanded =
		"flex h-10 items-center gap-2 truncate px-3 text-sm";
	const collapsed =
		"flex h-10 w-8 items-center justify-center";
	if (props.to) {
		return (
			<Link
				to={props.to}
				activeProps={{ className: "bg-accent text-foreground" }}
				className={`${common} ${props.collapsed ? collapsed : expanded}`}
				onClick={props.onClick}
			>
				{props.icon}
				{!props.collapsed && <span>{props.label}</span>}
				{props.collapsed && <span className="sr-only">{props.label}</span>}
			</Link>
		);
	}
	return (
		<button
			type="button"
			onClick={props.onClick}
			className={`${common} ${props.collapsed ? collapsed : expanded}`}
			aria-label={props.label}
		>
			{props.icon}
			{!props.collapsed && <span>{props.label}</span>}
		</button>
	);
}

type SidebarProps = {
	className?: string;
};

const MAX_REPOS = 100;
const REPO_OFFSET = 0;
const FIVE_ITEM_MAX_H = "max-h-[13.5rem]";

export function Sidebar({ className }: SidebarProps) {
	const { t } = useTranslation();
	const [open, setOpen] = useState(true);
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
	}) => {
		if (!(data.docDirectory && data.codebaseDirectory)) {
			alert(t("home.selectBothDirectories"));
			return;
		}
		if (!data.name) {
			alert(t("home.projectNameRequired"));
			return;
		}

		try {
			await LinkRepositories(
				data.name,
				data.docDirectory,
				data.codebaseDirectory
			);
			alert(t("home.linkSuccess"));
			setIsAddProjectOpen(false);
			loadProjects();
		} catch (error) {
			const errorMsg = error instanceof Error ? error.message : String(error);
			if (errorMsg.startsWith("missing_git_repo")) {
				const dir = errorMsg.endsWith("documentation")
					? t("projectManager.docDirectory")
					: t("projectManager.codebaseDirectory");
				if (window.confirm(`${dir} + ${t("home.unexistantGitRepoCreate")}`)) {
					try {
						await Init(dir);
						await LinkRepositories(
							data.name,
							data.docDirectory,
							data.codebaseDirectory
						);
						alert(t("home.linkSuccess"));
						setIsAddProjectOpen(false);
						loadProjects();
					} catch (initError) {
						console.error("Error initializing git repo:", initError);
						alert(t("home.initGitError"));
					}
					return;
				}
				return;
			}
			console.error("Error linking repositories:", error);
			alert(t("home.linkError"));
			return;
		}
	};

	const base =
		"shrink-0 border-r bg-muted/20 p-4 overflow-x-visible overflow-y-hidden transition-[width] duration-200 ease-in-out flex flex-col";
	const width = open ? "w-64" : "w-16";
	const cn = [base, width, className].filter(Boolean).join(" ");

	return (
		<aside className={cn}>
			<div className="relative mb-4 h-6">
				<button
					aria-label={
						open ? t("sidebar.collapseSidebar") : t("sidebar.expandSidebar")
					}
					className="-right-2 absolute top-0 rounded-full border bg-background p-1 text-foreground shadow hover:bg-accent"
					onClick={() => setOpen((v) => !v)}
					type="button"
				>
					{open ? <ChevronLeft size={16} /> : <ChevronRight size={16} />}
				</button>
			</div>

			<div className="min-h-0 flex-1 overflow-hidden">
				<nav className="space-y-1">
					<NavItem
						to="/"
						label={t("sidebar.home")}
						icon={<Home size={18} />}
						collapsed={!open}
					/>

					{/* Projects section */}
					{open ? (
						<div className="mt-4 px-2">
							<div className="flex items-center justify-between">
                                <div className="flex h-10 items-center gap-2 truncate px-1 text-sm">
                                    <FolderKanban size={18} />
                                    <span>{t("sidebar.projects")}</span>
                                </div>
								<Button
									aria-label={t("home.addProject")}
									className="h-6 w-6 border-sidebar-border bg-sidebar text-sidebar-foreground shadow-xs hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
									onClick={() => setIsAddProjectOpen(true)}
									size="icon"
									variant="outline"
								>
									<Plus className="h-3 w-3" />
								</Button>
							</div>

							<ul
								className={`mt-1 space-y-1 overflow-y-auto pr-1 ${FIVE_ITEM_MAX_H}`}
							>
								{loading && (
									<li className="px-3 py-2 text-muted-foreground text-xs">
										{t("sidebar.loading")}
									</li>
								)}
								{!loading && projects.length === 0 && (
									<li className="px-3 py-2 text-muted-foreground text-xs">
										{t("sidebar.noProjects")}
									</li>
								)}
								{!loading &&
									projects.map((p) => {
										const projectId = String(p.ID);
										return (
											<li key={`${projectId}-${p.ProjectName}`}>
												<Link
													className="flex h-10 items-center truncate rounded px-3 text-foreground/80 text-sm hover:bg-accent hover:text-foreground"
													params={{ projectId }}
													to="/projects/$projectId"
												>
													{p.ProjectName}
												</Link>
											</li>
										);
									})}
							</ul>
						</div>
					) : (
						<NavItem
							label={t("sidebar.projects")}
							icon={<FolderKanban size={18} />}
							collapsed
							onClick={() => setOpen(true)}
						/>
					)}
				</nav>
			</div>

			<div className="mt-auto pt-4">
				<NavItem
					to="/settings"
					label={t("sidebar.settings")}
					icon={<Settings size={18} />}
					collapsed={!open}
				/>
			</div>

			<AddProjectDialog
				onClose={() => setIsAddProjectOpen(false)}
				onSubmit={handleAddProject}
				open={isAddProjectOpen}
			/>
		</aside>
	);
}
