import type { models } from "@go/models";
import {
	ArrowRight,
	ArrowRightLeft,
	CheckIcon,
	ChevronsUpDownIcon,
} from "lucide-react";
import { useId } from "react";
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

interface BranchSelectorProps {
	branches: models.BranchInfo[];
	sourceBranch: string | undefined;
	setSourceBranch: (branch: string) => void;
	targetBranch: string | undefined;
	setTargetBranch: (branch: string) => void;
	sourceOpen: boolean;
	setSourceOpen: (open: boolean) => void;
	targetOpen: boolean;
	setTargetOpen: (open: boolean) => void;
	swapBranches: () => void;
	disableControls: boolean;
}

export const BranchSelector = ({
	branches,
	sourceBranch,
	setSourceBranch,
	targetBranch,
	setTargetBranch,
	sourceOpen,
	setSourceOpen,
	targetOpen,
	setTargetOpen,
	swapBranches,
	disableControls,
}: BranchSelectorProps) => {
	const { t } = useTranslation();
	const sourceBranchComboboxId = useId();
	const sourceBranchListId = useId();
	const targetBranchComboboxId = useId();
	const targetBranchListId = useId();

	const canSwap = Boolean(sourceBranch && targetBranch);

	return (
		<div className="flex items-center gap-3">
			<div className="flex-1 space-y-2 rounded-lg border border-border/50 bg-muted/30 p-3">
				<Label
					className="text-muted-foreground text-xs"
					htmlFor={sourceBranchComboboxId}
				>
					{t("common.sourceBranch")}
				</Label>
				<Popover modal={true} onOpenChange={setSourceOpen} open={sourceOpen}>
					<PopoverTrigger asChild>
						<Button
							aria-controls={sourceBranchListId}
							aria-expanded={sourceOpen}
							className={cn(
								"w-full justify-between overflow-hidden hover:text-foreground",
								twTrigger
							)}
							disabled={disableControls}
							id={sourceBranchComboboxId}
							role="combobox"
							type="button"
							variant="outline"
						>
							<span className="min-w-0 flex-1 truncate text-left">
								{sourceBranch ?? t("common.sourceBranch")}
							</span>
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
							<CommandList className="max-h-[200px]" id={sourceBranchListId}>
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

			<div className="flex shrink-0 flex-col items-center gap-1">
				<ArrowRight className="h-5 w-5 text-muted-foreground" />
				<Button
					aria-label={t("common.swapBranches")}
					className="h-7 w-7 p-0"
					disabled={disableControls || !canSwap}
					onClick={swapBranches}
					type="button"
					variant="ghost"
				>
					<ArrowRightLeft className="h-3.5 w-3.5" />
				</Button>
			</div>

			<div className="flex-1 space-y-2 rounded-lg border border-border/50 bg-accent/30 p-3">
				<Label
					className="text-muted-foreground text-xs"
					htmlFor={targetBranchComboboxId}
				>
					{t("common.targetBranch")}
				</Label>
				<Popover modal={true} onOpenChange={setTargetOpen} open={targetOpen}>
					<PopoverTrigger asChild>
						<Button
							aria-controls={targetBranchListId}
							aria-expanded={targetOpen}
							className={cn(
								"w-full justify-between overflow-hidden hover:text-foreground",
								twTrigger
							)}
							disabled={disableControls}
							id={targetBranchComboboxId}
							role="combobox"
							type="button"
							variant="outline"
						>
							<span className="min-w-0 flex-1 truncate text-left">
								{targetBranch ?? t("common.targetBranch")}
							</span>
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
							<CommandList className="max-h-[200px]" id={targetBranchListId}>
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
		</div>
	);
};
