import type { models } from "@go/models";
import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";
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
							<TooltipProvider>
								<Tooltip>
									<TooltipTrigger asChild>
										<div>
											<Button
												className="gap-2 border-border text-foreground hover:bg-accent disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
												disabled={!canMerge}
												onClick={onMerge}
												variant="outline"
											>
												{isMerging
													? t("common.mergingDocs", "Merging…")
													: t("common.mergeDocsIntoSource", "Merge into source branch")}
											</Button>
										</div>
									</TooltipTrigger>
									{!canMerge && mergeDisabledReason && (
										<TooltipContent>
											{mergeDisabledReason === "onSourceBranchWithUncommitted" ? (
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
						)}
						<Button
							className="gap-2 font-semibold disabled:cursor-not-allowed disabled:border disabled:border-border disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
							disabled={!canCommit}
							onClick={onCommit}
						>
							{t("common.commit")}
							<ArrowRight className="h-4 w-4" />
						</Button>
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
