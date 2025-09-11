import { ArrowRight } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
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
import { GetRepoLinks, ListRepoBranches } from "../../../wailsjs/go/main/App";
import type { models } from "../../../wailsjs/go/models";

const twTrigger =
	"h-10 w-full bg-card text-card-foreground border border-border " +
	"hover:bg-muted data-[state=open]:bg-muted " +
	"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40";
const twContent =
	"bg-popover text-popover-foreground border border-border shadow-md";
const twItem =
	"data-[highlighted]:bg-muted data-[highlighted]:text-foreground " +
	"data-[state=checked]:bg-primary/15 data-[state=checked]:text-foreground";

function GenerateDocsDialog({
	open,
	onClose,
}: {
	open: boolean;
	onClose: () => void;
}) {
	const { t } = useTranslation();

	const [projects, setProjects] = useState<models.RepoLink[]>([]);
	const [selectedProject, setSelectedProject] = useState<
		models.RepoLink | undefined
	>();

	const [branches, setBranches] = useState<string[]>([]);
	const [sourceBranch, setSourceBranch] = useState<string | undefined>();
	const [targetBranch, setTargetBranch] = useState<string | undefined>();

	useEffect(() => {
		if (open) {
			GetRepoLinks()
				.then((repos) => setProjects(repos))
				.catch((err) => console.error("failed to fetch repo links:", err));
		}
	}, [open]);

	useEffect(() => {
		if (selectedProject) {
			ListRepoBranches(selectedProject.CodebaseRepo)
				.then(setBranches)
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
					{/* Project select */}
					<div className="grid gap-2">
						<Label className="mb-1" htmlFor="project-select">
							{t("common.project")}
						</Label>
						<Select
							onValueChange={(id) => {
								const proj = projects.find((p) => p.ID.toString() === id);
								setSelectedProject(proj);
							}}
							value={selectedProject?.ID.toString()}
						>
							<SelectTrigger className={twTrigger} id="project-select">
								<SelectValue placeholder={t("common.selectProject")} />
							</SelectTrigger>
							<SelectContent className={twContent}>
								{projects.map((p) => (
									<SelectItem
										className={twItem}
										key={p.ID}
										value={p.ID.toString()}
									>
										{p.ProjectName}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* Branch selects */}
					{selectedProject && (
						<>
							<div className="grid gap-2">
								<div className="flex items-center justify-between">
									<Label className="mb-1" htmlFor="source-branch-select">
										{t("common.branchSource")}
									</Label>
									<Button
										onClick={swapBranches}
										size="sm"
										type="button"
										variant="secondary"
									>
										{t("common.swapBranches")}
									</Button>
								</div>
								<Select onValueChange={setSourceBranch} value={sourceBranch}>
									<SelectTrigger
										className={twTrigger}
										id="source-branch-select"
									>
										<SelectValue placeholder={t("common.branchSource")} />
									</SelectTrigger>
									<SelectContent className={twContent}>
										{branches.map((b) => (
											<SelectItem className={twItem} key={b} value={b}>
												{b}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>

							<div className="grid gap-2">
								<Label className="mb-1" htmlFor="target-branch-select">
									{t("common.branchTarget")}
								</Label>
								<Select onValueChange={setTargetBranch} value={targetBranch}>
									<SelectTrigger
										className={twTrigger}
										id="target-branch-select"
									>
										<SelectValue placeholder={t("common.branchTarget")} />
									</SelectTrigger>
									<SelectContent className={twContent}>
										{branches.map((b) => (
											<SelectItem className={twItem} key={b} value={b}>
												{b}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						</>
					)}
				</div>

				<DialogFooter className="mt-2">
					<Button
						className="border-border text-foreground hover:bg-muted"
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

export { GenerateDocsDialog };
