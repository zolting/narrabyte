import {
	closestCenter,
	DndContext,
	type DragEndEvent,
	KeyboardSensor,
	PointerSensor,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import {
	arrayMove,
	SortableContext,
	sortableKeyboardCoordinates,
	useSortable,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { models } from "@go/models";
import { Init } from "@go/services/GitService";
import {
	Delete,
	LinkRepositories,
	List,
	UpdateProjectOrder,
} from "@go/services/repoLinkService";
import { Link, useLocation, useNavigate } from "@tanstack/react-router";
import {
	FileText,
	Folder,
	Folders,
	GripVertical,
	Home,
	Plus,
	Search,
	Settings,
	Trash2,
} from "lucide-react";
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
import { Input } from "@/components/ui/input";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarTrigger,
} from "@/components/ui/sidebar";

const MAX_REPOS = 100;
const REPO_OFFSET = 0;

function SortableProjectItem({
	project,
	isActive,
	onDelete,
	onNavigateToSettings,
}: {
	project: models.RepoLink;
	isActive: boolean;
	onDelete: () => void;
	onNavigateToSettings: () => void;
}) {
	const navigate = useNavigate();
	const { t } = useTranslation();
	const projectId = String(project.ID);

	const {
		attributes,
		listeners,
		setNodeRef,
		transform,
		transition,
		isDragging,
	} = useSortable({ id: projectId });

	const style = {
		transform: CSS.Transform.toString(transform),
		transition,
		opacity: isDragging ? 0.3 : 1,
		boxShadow: isDragging ? "0 4px 12px rgba(0, 0, 0, 0.15)" : "none",
		scale: isDragging ? 1.02 : 1,
	};

	return (
		<SidebarMenuItem ref={setNodeRef} style={style}>
			<ContextMenu>
				<ContextMenuTrigger>
					<div className="flex w-full items-center gap-1">
						<button
							className="cursor-grab rounded p-1 text-muted-foreground/60 hover:bg-sidebar-accent hover:text-muted-foreground active:cursor-grabbing group-data-[collapsible=icon]:hidden"
							type="button"
							{...attributes}
							{...listeners}
						>
							<GripVertical size={16} />
						</button>
						<SidebarMenuButton
							asChild
							className="flex-1"
							isActive={isActive}
							size="sm"
							tooltip={project.ProjectName}
						>
							<Link params={{ projectId }} to="/projects/$projectId">
								<Folder size={16} />
								<span className="text-sm">{project.ProjectName}</span>
							</Link>
						</SidebarMenuButton>
					</div>
				</ContextMenuTrigger>
				<ContextMenuContent>
					<ContextMenuItem onSelect={onNavigateToSettings}>
						<Settings size={16} />
						<span>{t("sidebar.projectSettings")}</span>
					</ContextMenuItem>
					<ContextMenuItem
						onSelect={() => {
							navigate({
								to: "/projects/$projectId/generations",
								params: { projectId },
							});
						}}
					>
						<FileText size={16} />
						<span>{t("sidebar.ongoingGenerations")}</span>
					</ContextMenuItem>
					<ContextMenuItem onSelect={onDelete} variant="destructive">
						<Trash2 size={16} />
						<span>{t("sidebar.deleteProject")}</span>
					</ContextMenuItem>
				</ContextMenuContent>
			</ContextMenu>
		</SidebarMenuItem>
	);
}

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
	const [searchQuery, setSearchQuery] = useState("");

	const sensors = useSensors(
		useSensor(PointerSensor, {
			activationConstraint: {
				distance: 8,
			},
		}),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		})
	);

	const loadProjects = useCallback(() => {
		setLoading(true);
		Promise.resolve(List(MAX_REPOS, REPO_OFFSET))
			.then((res) => {
				const sorted = ((res as models.RepoLink[]) ?? []).sort((a, b) => {
					const indexA = a.index ?? a.ID;
					const indexB = b.index ?? b.ID;
					return indexA - indexB;
				});
				setProjects(sorted);
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

	const handleDragEnd = (event: DragEndEvent) => {
		const { active, over } = event;

		if (!over || active.id === over.id) {
			return;
		}

		setProjects((items) => {
			const oldIndex = items.findIndex((item) => String(item.ID) === active.id);
			const newIndex = items.findIndex((item) => String(item.ID) === over.id);

			const newOrder = arrayMove(items, oldIndex, newIndex);

			updateProjectOrder(newOrder).catch((error) => {
				console.error("Error updating project order:", error);
				toast(t("sidebar.reorderError"));
				loadProjects();
			});

			return newOrder;
		});
	};

	const updateProjectOrder = async (orderedProjects: models.RepoLink[]) => {
		const updates = orderedProjects.map(
			(project, index) =>
				new models.RepoLinkOrderUpdate({
					ID: project.ID,
					Index: index,
				})
		);

		await UpdateProjectOrder(updates);
	};

	const filteredProjects = projects.filter((project) =>
		project.ProjectName.toLowerCase().includes(searchQuery.toLowerCase())
	);

	return (
		<>
			<SidebarHeader className="border-sidebar-border border-b p-2">
				<SidebarTrigger />
			</SidebarHeader>

			<SidebarContent className="pt-2">
				<SidebarGroup className="pb-2">
					<SidebarMenu>
						<SidebarMenuItem>
							<SidebarMenuButton
								asChild
								isActive={location.pathname === "/"}
								size="default"
								tooltip={t("sidebar.home")}
							>
								<Link to="/">
									<Home size={16} />
									<span>{t("sidebar.home")}</span>
								</Link>
							</SidebarMenuButton>
						</SidebarMenuItem>
					</SidebarMenu>
				</SidebarGroup>
				<SidebarGroup className="border-sidebar-border border-t pt-2">
					<div className="mb-2 space-y-2">
						<div className="flex items-center justify-between px-2">
							<SidebarGroupLabel className="mb-0 flex-1 p-0 font-semibold text-sidebar-foreground">
								<Folders size={16} />
								<span className="ml-1">{t("sidebar.projects")}</span>
							</SidebarGroupLabel>
							<Button
								aria-label={t("home.addProject")}
								className="h-5 w-5 p-0 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground group-data-[collapsible=icon]:hidden"
								onClick={() => setIsAddProjectOpen(true)}
								size="sm"
								variant="ghost"
							>
								<Plus className="h-3.5 w-3.5" />
							</Button>
						</div>
						{projects.length > 5 && (
							<div className="px-2 group-data-[collapsible=icon]:hidden">
								<div className="relative">
									<Search
										className="-translate-y-1/2 pointer-events-none absolute top-1/2 left-2.5 text-muted-foreground"
										size={14}
									/>
									<Input
										className="h-7 pr-2 pl-8 text-xs"
										onChange={(e) => setSearchQuery(e.target.value)}
										placeholder={t("sidebar.searchProjects")}
										type="search"
										value={searchQuery}
									/>
									{searchQuery && (
										<button
											className="-translate-y-1/2 absolute top-1/2 right-2 rounded-sm text-muted-foreground hover:text-foreground"
											onClick={() => setSearchQuery("")}
											type="button"
										/>
									)}
								</div>
							</div>
						)}
					</div>
					<SidebarGroupContent>
						<SidebarMenu>
							{loading &&
								[1, 2, 3].map((i) => (
									<SidebarMenuItem key={i}>
										<div className="flex items-center gap-2 px-2 py-1.5 group-data-[collapsible=icon]:hidden">
											<div className="h-4 w-4 animate-pulse rounded bg-sidebar-accent" />
											<div className="h-4 flex-1 animate-pulse rounded bg-sidebar-accent" />
										</div>
									</SidebarMenuItem>
								))}
							{!loading && projects.length === 0 && (
								<SidebarMenuItem>
									<div className="flex flex-col gap-2 px-2 py-4 text-center group-data-[collapsible=icon]:hidden">
										<Folder
											className="mx-auto text-muted-foreground"
											size={32}
										/>
										<p className="text-muted-foreground text-xs">
											{t("sidebar.noProjects")}
										</p>
										<Button
											className="mt-1"
											onClick={() => setIsAddProjectOpen(true)}
											size="sm"
											variant="outline"
										>
											<Plus size={16} />
											{t("home.addProject")}
										</Button>
									</div>
								</SidebarMenuItem>
							)}
							{!loading &&
								projects.length > 0 &&
								filteredProjects.length === 0 && (
									<SidebarMenuItem>
										<div className="flex flex-col gap-2 px-2 py-4 text-center group-data-[collapsible=icon]:hidden">
											<Search
												className="mx-auto text-muted-foreground"
												size={32}
											/>
											<p className="text-muted-foreground text-xs">
												{t("sidebar.noProjectsFound", "No projects found")}
											</p>
										</div>
									</SidebarMenuItem>
								)}
							{!loading && filteredProjects.length > 0 && (
								<DndContext
									collisionDetection={closestCenter}
									onDragEnd={handleDragEnd}
									sensors={sensors}
								>
									<SortableContext
										items={filteredProjects.map((p) => String(p.ID))}
										strategy={verticalListSortingStrategy}
									>
										{filteredProjects.map((p) => {
											const projectId = String(p.ID);
											return (
												<SortableProjectItem
													isActive={
														location.pathname === `/projects/${projectId}`
													}
													key={projectId}
													onDelete={() => openDeleteDialog(p)}
													onNavigateToSettings={() => {
														navigate({
															to: "/projects/$projectId/settings",
															params: { projectId },
														});
													}}
													project={p}
												/>
											);
										})}
									</SortableContext>
								</DndContext>
							)}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>

			<SidebarFooter className="border-sidebar-border border-t p-2">
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton
							asChild
							isActive={location.pathname === "/settings"}
							size="default"
							tooltip={t("sidebar.settings")}
						>
							<Link to="/settings">
								<Settings size={16} />
								<span>{t("sidebar.settings")}</span>
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
						<AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
						<AlertDialogAction
							className="bg-destructive text-destructive-foreground hover:brightness-90 dark:hover:brightness-110"
							onClick={handleDeleteProject}
						>
							{t("common.delete")}
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
