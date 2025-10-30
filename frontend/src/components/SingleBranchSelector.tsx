import type { models } from "@go/models";
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react";
import { useId, useMemo } from "react";
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

export function SingleBranchSelector({
	branches,
	branch,
	setBranch,
	open,
	setOpen,
	disableControls,
}: {
	branches: models.BranchInfo[];
	branch: string | undefined;
	setBranch: (b: string) => void;
	open: boolean;
	setOpen: (o: boolean) => void;
	disableControls: boolean;
}) {
	const { t } = useTranslation();
	const branchComboboxId = useId();
	const branchListId = useId();

	const items = useMemo(() => branches, [branches]);

	return (
		<div className="min-w-0 space-y-2 rounded-lg border border-border/50 bg-muted/30 p-3">
			<Label
				className="text-muted-foreground text-xs"
				htmlFor={branchComboboxId}
			>
				{t("common.sourceBranch")}
			</Label>
			<Popover modal={true} onOpenChange={setOpen} open={open}>
				<PopoverTrigger asChild>
					<Button
						aria-controls={branchListId}
						aria-expanded={open}
						className={cn(
							"w-full justify-between overflow-hidden hover:text-foreground",
							twTrigger
						)}
						disabled={disableControls}
						id={branchComboboxId}
						role="combobox"
						type="button"
						variant="outline"
					>
						<span className="min-w-0 flex-1 truncate text-left">
							{branch ?? t("common.sourceBranch")}
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
						<CommandList className="max-h-[200px]" id={branchListId}>
							<CommandEmpty>
								{t("common.noBranchFound", "No branch found.")}
							</CommandEmpty>
							<CommandGroup>
								{items.map((b) => (
									<CommandItem
										key={b.name}
										onSelect={(val) => {
											setBranch(val);
											setOpen(false);
										}}
										value={b.name}
									>
										<CheckIcon
											className={cn(
												"mr-2 h-4 w-4",
												branch === b.name ? "opacity-100" : "opacity-0"
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
	);
}
