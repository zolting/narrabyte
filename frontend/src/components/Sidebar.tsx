import type { models } from "@go/models";
import { Init } from "@go/services/GitService";
import { LinkRepositories, List } from "@go/services/repoLinkService";
import { Link } from "@tanstack/react-router";
import { ChevronLeft, ChevronRight, Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import AddProjectDialog from "@/components/AddProjectDialog";
import { Button } from "@/components/ui/button";

function NavItem(props: { to: string; label: string }) {
	return (
		<Link
			activeProps={{ className: "bg-accent text-foreground" }}
			className="block rounded px-3 py-2 text-foreground/80 text-sm hover:bg-accent hover:text-foreground"
			to={props.to}
		>
			{props.label}
		</Link>
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

	const loadProjects = () => {
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
	};

	useEffect(() => {
		loadProjects();
	}, []);

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
				const which = errorMsg.endsWith("documentation")
					? t("projectManager.docDirectory")
					: t("projectManager.codebaseDirectory");
				if (window.confirm(`${which} + ${t("home.unexistantGitRepoCreate")}`)) {
					const dir = errorMsg.endsWith("documentation")
						? data.docDirectory
						: data.codebaseDirectory;

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
	const width = open ? "w-64" : "w-10";
	const cn = [base, width, className].filter(Boolean).join(" ");

	return (
		<aside className={cn}>
			{/* Open/close button */}
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

			{/* Top content */}
			<div className={open ? "min-h-0 flex-1 overflow-hidden" : "hidden"}>
				<nav className="space-y-1">
					<NavItem label={t("sidebar.home")} to="/" />

					<div className="mt-4 flex items-center justify-between px-2">
						<div className="font-semibold text-muted-foreground text-xs uppercase tracking-wide">
							{t("sidebar.projects")}
						</div>
						<Button
							aria-label={t("home.addProject")}
							className="h-6 w-6"
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
				</nav>
			</div>

			{/* Bottom content */}
			<div className={open ? "mt-auto pt-4" : "hidden"}>
				<NavItem label={t("sidebar.settings")} to="/settings" />
			</div>

			<AddProjectDialog
				onClose={() => setIsAddProjectOpen(false)}
				onSubmit={handleAddProject}
				open={isAddProjectOpen}
			/>
		</aside>
	);
}
