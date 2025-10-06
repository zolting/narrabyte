import type { models } from "@go/models";
import { ArrowRight } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
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
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "@/components/ui/tooltip";

interface ActionButtonsProps {
	docResult: models.DocGenerationResult | null;
	isRunning: boolean;
	isBusy: boolean;
	canGenerate: boolean;
	canCommit: boolean;
	canMerge: boolean;
	isMerging: boolean;
	docGenerationError: string | null;
	mergeDisabledReason: string | null;
	onCancel: () => void;
	onReset: () => void;
	onCommit: () => void;
	onGenerate: () => void;
	onMerge: () => void;
}

export const ActionButtons = ({
	docResult,
	isRunning,
	isBusy,
	canGenerate,
	canCommit,
	canMerge,
	isMerging,
	docGenerationError,
	mergeDisabledReason,
	onCancel,
	onReset,
	onCommit,
	onGenerate,
	onMerge,
}: ActionButtonsProps) => {
	const { t } = useTranslation();
	const [showMergeConfirm, setShowMergeConfirm] = useState(false);
	const [showApproveConfirm, setShowApproveConfirm] = useState(false);

	return (
		<footer className="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
			{docGenerationError && (
				<div className="text-destructive text-xs">{docGenerationError}</div>
			)}
			<div className="flex items-center gap-2 sm:justify-end">
				{isRunning && (
					<Button
						className="font-semibold"
						onClick={onCancel}
						type="button"
						variant="destructive"
					>
						{t("common.cancel")}
					</Button>
				)}
				<Button
					className="border-border text-foreground hover:bg-accent"
					disabled={isBusy}
					onClick={onReset}
					variant="outline"
				>
					{t("common.reset")}
				</Button>
				{docResult ? (
					<>
						{docResult.docsInCodeRepo && (
							<>
								<TooltipProvider>
									<Tooltip>
										<TooltipTrigger asChild>
											<div>
												<Button
													className="gap-2 border-border text-foreground hover:bg-accent disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
													disabled={!canMerge}
													onClick={() => setShowMergeConfirm(true)}
													variant="outline"
												>
													{isMerging
														? t("common.mergingDocs", "Merging…")
														: t(
																"common.mergeDocsIntoSource",
																"Merge into source branch"
															)}
												</Button>
											</div>
										</TooltipTrigger>
										{!canMerge && mergeDisabledReason && (
											<TooltipContent>
												{mergeDisabledReason ===
												"onSourceBranchWithUncommitted" ? (
													<p className="max-w-xs text-xs">
														{t(
															"common.mergeDisabledUncommittedChanges",
															"Cannot merge: You are currently on the source branch with uncommitted changes. Please commit or stash your changes first."
														)}
													</p>
												) : null}
											</TooltipContent>
										)}
									</Tooltip>
								</TooltipProvider>
								<AlertDialog
									onOpenChange={setShowMergeConfirm}
									open={showMergeConfirm}
								>
									<AlertDialogContent>
										<AlertDialogHeader>
											<AlertDialogTitle>
												{t(
													"common.confirmMergeTitle",
													"Merge documentation into source branch?"
												)}
											</AlertDialogTitle>
											<AlertDialogDescription>
												{t(
													"common.confirmMergeDescription",
													"This will fast-forward your source branch ({branch}) to include the documentation commit. The changes will be immediately available on {branch}.",
													{ branch: docResult.branch }
												)}
											</AlertDialogDescription>
										</AlertDialogHeader>
										<AlertDialogFooter>
											<AlertDialogCancel>
												{t("common.cancel", "Cancel")}
											</AlertDialogCancel>
											<AlertDialogAction
												onClick={() => {
													setShowMergeConfirm(false);
													onMerge();
												}}
											>
												{t("common.merge", "Merge")}
											</AlertDialogAction>
										</AlertDialogFooter>
									</AlertDialogContent>
								</AlertDialog>
							</>
						)}
						<Button
							className="gap-2 font-semibold disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
							disabled={!canCommit}
							onClick={() => setShowApproveConfirm(true)}
						>
							{t("common.approve")}
							<ArrowRight className="h-4 w-4" />
						</Button>
						<AlertDialog
							onOpenChange={setShowApproveConfirm}
							open={showApproveConfirm}
						>
							<AlertDialogContent>
								<AlertDialogHeader>
									<AlertDialogTitle>
										{t("common.confirmApprovalTitle")}
									</AlertDialogTitle>
									<AlertDialogDescription>
										{t("common.confirmApprovalDescription")}
									</AlertDialogDescription>
								</AlertDialogHeader>
								<AlertDialogFooter>
									<AlertDialogCancel>
										{t("common.cancel")}
									</AlertDialogCancel>
									<AlertDialogAction
										onClick={() => {
											setShowApproveConfirm(false);
											onCommit();
										}}
									>
										{t("common.approve")}
									</AlertDialogAction>
								</AlertDialogFooter>
							</AlertDialogContent>
						</AlertDialog>
					</>
				) : (
					<Button
						className="gap-2 font-semibold disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
						disabled={!canGenerate}
						onClick={onGenerate}
					>
						{t("common.generateDocs")}
						<ArrowRight className="h-4 w-4" />
					</Button>
				)}
			</div>
		</footer>
	);
};
