import type { models } from "@go/models";
import { ListBranchesByPath } from "@go/services/GitService";
import { ArrowRight, CheckIcon, ChevronsUpDownIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
	Command,
	CommandEmpty,
	CommandGroup,
	CommandInput,
	CommandItem,
	CommandList,
} from "@/components/ui/command";
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
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";

const twTrigger =
	"h-10 w-full bg-card text-card-foreground border border-border " +
	"hover:bg-muted data-[state=open]:bg-muted " +
	"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40";
const twContent =
	"bg-popover text-popover-foreground border border-border shadow-md";

export default function GenerateDocsDialog({
	open,
	onClose,
	project,
}: {
	open: boolean;
	onClose: () => void;
	project: models.RepoLink;
}) {
	const { t } = useTranslation();

	const [selectedProject, setSelectedProject] = useState<
		models.RepoLink | undefined
	>();

	const [branches, setBranches] = useState<models.BranchInfo[]>([]);
	const [sourceBranch, setSourceBranch] = useState<string | undefined>();
	const [targetBranch, setTargetBranch] = useState<string | undefined>();
	const [sourceOpen, setSourceOpen] = useState(false);
	const [targetOpen, setTargetOpen] = useState(false);

	useEffect(() => {
		if (!open) {
			return;
		}
		setSelectedProject(project);
	}, [open, project]);

	useEffect(() => {
		if (selectedProject) {
			ListBranchesByPath(selectedProject.CodebaseRepo)
				.then((arr) =>
					setBranches(
						[...arr].sort(
							(a, b) =>
								new Date(b.lastCommitDate as unknown as string).getTime() -
								new Date(a.lastCommitDate as unknown as string).getTime()
						)
					)
				)
				.catch((err) => console.error("failed to fetch branches:", err));
		} else {
			setBranches([]);
			setSourceBranch(undefined);
			setTargetBranch(undefined);
		}
	}, [selectedProject]);

	const canContinue = useMemo(
		() =>
			Boolean(
				selectedProject &&
					sourceBranch &&
					targetBranch &&
					sourceBranch !== targetBranch
			),
		[selectedProject, sourceBranch, targetBranch]
	);

	const swapBranches = () => {
		setSourceBranch((s) => {
			const next = targetBranch;
			setTargetBranch(s);
			return next;
		});
	};

	const handleOpenChange = useCallback(
		(isOpen: boolean) => {
			if (!isOpen) {
				onClose();
			}
		},
		[onClose]
	);

	return (
		<Dialog onOpenChange={handleOpenChange} open={open}>
			<DialogContent className="sm:max-w-[520px]">
				<DialogHeader>
					<DialogTitle className="font-semibold text-lg text-primary sm:text-xl">
						{t("common.generateDocs")}
					</DialogTitle>
					<DialogDescription>
						{t("common.generateDocsDescription")}
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-6">
					{/* Project (read-only) */}
					<div className="grid gap-2">
						<Label className="mb-1 text-foreground" htmlFor="project-readonly">
							{t("common.project")}
						</Label>
						<div
							className={cn(
								"h-10 w-full rounded-md border border-border bg-card text-card-foreground",
								"flex items-center px-3"
							)}
							id="project-readonly"
						>
							{project.ProjectName}
						</div>
					</div>

					{/* Branch comboboxes */}
					{selectedProject && (
						<>
							<div className="grid gap-2">
								<div className="flex items-center justify-between">
									<Label
										className="mb-1 text-foreground"
										htmlFor="source-branch-combobox"
									>
										{t("common.sourceBranch")}
									</Label>
									<Button
										className="hover:bg-accent"
										onClick={swapBranches}
										size="sm"
										type="button"
										variant="secondary"
									>
										{t("common.swapBranches")}
									</Button>
								</div>
								<Popover
									modal={true}
									onOpenChange={setSourceOpen}
									open={sourceOpen}
								>
									<PopoverTrigger asChild>
										<Button
											aria-controls="source-branch-list"
											aria-expanded={sourceOpen}
											className={cn(
												"w-full justify-between hover:text-foreground",
												twTrigger
											)}
											id="source-branch-combobox"
											role="combobox"
											type="button"
											variant="outline"
										>
											{sourceBranch ?? t("common.sourceBranch")}
											<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
										</Button>
									</PopoverTrigger>
									<PopoverContent
										className={cn(
											"w-[var(--radix-popover-trigger-width)] p-0",
											twContent
										)}
									>
										<Command>
											<CommandInput placeholder="Search branch..." />
											<CommandList
												className="max-h-[200px]"
												id="source-branch-list"
											>
												<CommandEmpty>No branch found.</CommandEmpty>
												<CommandGroup>
													{branches
														.filter((b) => b.name !== targetBranch)
														.map((b) => (
															<CommandItem
																key={b.name}
																onSelect={(currentValue) => {
																	setSourceBranch(currentValue);
																	setSourceOpen(false);
																}}
																value={b.name}
															>
																<CheckIcon
																	className={cn(
																		"mr-2 h-4 w-4",
																		sourceBranch === b.name
																			? "opacity-100"
																			: "opacity-0"
																	)}
																/>
																{b.name}
															</CommandItem>
														))}
												</CommandGroup>
											</CommandList>
										</Command>
									</PopoverContent>
								</Popover>
							</div>

							<div className="grid gap-2">
								<Label
									className="mb-1 text-foreground"
									htmlFor="target-branch-combobox"
								>
									{t("common.targetBranch")}
								</Label>
								<Popover
									modal={true}
									onOpenChange={setTargetOpen}
									open={targetOpen}
								>
									<PopoverTrigger asChild>
										<Button
											aria-controls="target-branch-list"
											aria-expanded={targetOpen}
											className={cn(
												"w-full justify-between hover:text-foreground",
												twTrigger
											)}
											id="target-branch-combobox"
											role="combobox"
											type="button"
											variant="outline"
										>
											{targetBranch ?? t("common.targetBranch")}
											<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
										</Button>
									</PopoverTrigger>
									<PopoverContent
										className={cn(
											"w-[var(--radix-popover-trigger-width)] p-0",
											twContent
										)}
									>
										<Command>
											<CommandInput placeholder="Search branch..." />
											<CommandList
												className="max-h-[200px]"
												id="target-branch-list"
											>
												<CommandEmpty>No branch found.</CommandEmpty>
												<CommandGroup>
													{branches
														.filter((b) => b.name !== sourceBranch)
														.map((b) => (
															<CommandItem
																key={b.name}
																onSelect={(currentValue) => {
																	setTargetBranch(currentValue);
																	setTargetOpen(false);
																}}
																value={b.name}
															>
																<CheckIcon
																	className={cn(
																		"mr-2 h-4 w-4",
																		targetBranch === b.name
																			? "opacity-100"
																			: "opacity-0"
																	)}
																/>
																{b.name}
															</CommandItem>
														))}
												</CommandGroup>
											</CommandList>
										</Command>
									</PopoverContent>
								</Popover>
							</div>
						</>
					)}
				</div>

				<DialogFooter className="mt-2">
					<Button
						className="border-border text-foreground hover:bg-accent"
						onClick={onClose}
						variant="outline"
					>
						{t("common.cancel")}
					</Button>
					<Button
						className="gap-2 font-semibold disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
						disabled={!canContinue}
					>
						{t("common.continue")}
						<ArrowRight className="h-4 w-4" />
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
