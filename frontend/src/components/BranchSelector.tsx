import type { models } from "@go/models";
import { ArrowRightLeft, CheckIcon, ChevronsUpDownIcon } from "lucide-react";
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

	return (
		<div className="grid shrink-0 grid-cols-[1fr_auto_1fr] items-end gap-4">
			<div className="grid min-w-0 gap-2">
				<Label
					className="mb-1 text-foreground"
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

			<Button
				aria-label={t("common.swapBranches")}
				className="h-10 w-10 p-1 hover:bg-accent"
				disabled={disableControls || branches.length < 2}
				onClick={swapBranches}
				type="button"
				variant="secondary"
			>
				<ArrowRightLeft className="h-4 w-4" />
			</Button>

			<div className="grid min-w-0 gap-2">
				<Label
					className="mb-1 text-foreground"
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
