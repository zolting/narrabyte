import type { models } from "@go/models";
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react";
import { useId, useMemo, useState } from "react";
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
import { sortBranches } from "@/lib/sortBranches";
import { cn } from "@/lib/utils";

interface DocumentationBranchSelectorProps {
	branches: models.BranchInfo[];
	value: string;
	onChange: (branch: string) => void;
	disabled?: boolean;
	description?: string;
	className?: string;
}

const triggerClasses =
	"h-10 w-full justify-between overflow-hidden text-left bg-card text-card-foreground " +
	"border border-border hover:bg-muted data-[state=open]:bg-muted focus-visible:outline-none " +
	"focus-visible:ring-2 focus-visible:ring-primary/40";

export function DocumentationBranchSelector({
	branches,
	value,
	onChange,
	disabled = false,
	description,
	className,
}: DocumentationBranchSelectorProps) {
	const { t } = useTranslation();
	const [open, setOpen] = useState(false);
	const [search, setSearch] = useState("");
	const comboboxId = useId();
	const listId = useId();

	const orderedBranches = useMemo(
		() => sortBranches(branches, { prioritizeMainMaster: true }),
		[branches]
	);
	const trimmedSearch = search.trim();
	const hasBranches = orderedBranches.length > 0;
	const hasMatch =
		hasBranches && orderedBranches.some((b) => b.name === trimmedSearch);

	const handleSelect = (branch: string) => {
		onChange(branch);
		setOpen(false);
		setSearch("");
	};

	return (
		<div className={cn("space-y-2", className)}>
			<div className="flex flex-col gap-1">
				<Label className="text-sm" htmlFor={comboboxId}>
					{t("common.documentationBaseBranch")}
				</Label>
				{description && (
					<p className="text-muted-foreground text-xs">{description}</p>
				)}
			</div>
			<Popover modal={true} onOpenChange={setOpen} open={open}>
				<PopoverTrigger asChild>
					<Button
						aria-controls={listId}
						aria-expanded={open}
						className={triggerClasses}
						disabled={disabled}
						id={comboboxId}
						role="combobox"
						type="button"
						variant="outline"
					>
						<span className="min-w-0 flex-1 truncate">
							{value || t("common.documentationBaseBranchPlaceholder")}
						</span>
						<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
					</Button>
				</PopoverTrigger>
				<PopoverContent
					align="start"
					className="w-[var(--radix-popover-trigger-width)] p-0"
				>
					<Command shouldFilter={false}>
						<CommandInput
							onValueChange={setSearch}
							placeholder={t("common.searchBranchPlaceholder")}
							value={search}
						/>
						<CommandList className="max-h-[200px]" id={listId}>
							<CommandEmpty>
								{hasBranches
									? t("common.noMatchingBranches")
									: t("common.noBranchesFound")}
							</CommandEmpty>
							<CommandGroup>
								{orderedBranches
									.filter((branch) => {
										if (!trimmedSearch) {
											return true;
										}
										const target = branch.name?.toLowerCase() ?? "";
										return target.includes(trimmedSearch.toLowerCase());
									})
									.map((branch) => (
										<CommandItem
											key={branch.name}
											onSelect={handleSelect}
											value={branch.name ?? ""}
										>
											<CheckIcon
												className={cn(
													"mr-2 h-4 w-4",
													value === branch.name ? "opacity-100" : "opacity-0"
												)}
											/>
											{branch.name}
										</CommandItem>
									))}
							</CommandGroup>
							{trimmedSearch && !hasMatch && (
								<CommandGroup heading={t("common.customBranchHeader")}>
									<CommandItem
										onSelect={() => handleSelect(trimmedSearch)}
										value={trimmedSearch}
									>
										{t("common.useBranchOption", {
											branch: trimmedSearch,
										})}
									</CommandItem>
								</CommandGroup>
							)}
						</CommandList>
					</Command>
				</PopoverContent>
			</Popover>
		</div>
	);
}
