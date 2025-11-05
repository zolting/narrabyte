import {
	AlertCircle,
	CheckIcon,
	ChevronsUpDownIcon,
	EditIcon,
	Trash,
} from "lucide-react";
import type { KeyboardEvent } from "react";
import { useCallback, useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
	Command,
	CommandEmpty,
	CommandGroup,
	CommandInput,
	CommandItem,
	CommandList,
	CommandSeparator,
} from "@/components/ui/command";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { useTemplateStore } from "@/stores/template";

interface TemplateSelectorProps {
	setTemplateInstructions: (instructions: string) => void;
}

const triggerClasses =
	"h-10 w-full justify-between overflow-hidden bg-card text-card-foreground border border-border hover:bg-muted data-[state=open]:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40";
const contentClasses =
	"bg-popover text-popover-foreground border border-border shadow-md";

export const TemplateSelector = ({
	setTemplateInstructions,
}: TemplateSelectorProps) => {
	const { t } = useTranslation();
	const comboboxId = useId();
	const listId = useId();

	const [selectedTemplate, setSelectedTemplate] = useState<string | undefined>(
		undefined
	);
	const editTemplate = useTemplateStore((state) => state.editTemplate);
	const deleteTemplate = useTemplateStore((state) => state.deleteTemplate);
	const loadTemplates = useTemplateStore((state) => state.loadTemplates);
	const templates = useTemplateStore((state) => state.templates);

	const error = useTemplateStore((state) => state.error);
	const clearError = useTemplateStore((state) => state.clearError);

	const [open, setOpen] = useState(false);
	const [editOpen, setEditOpen] = useState(false);
	const [deleteOpen, setDeleteOpen] = useState(false);

	const [currentName, setCurrentName] = useState<string | undefined>(
		selectedTemplate
	);

	const [editingName, setEditingName] = useState("");
	const [editingContent, setEditingContent] = useState("");
	const [editTargetId, setEditTargetId] = useState<number | null>(null);
	const [deleteTargetName, setDeleteTargetName] = useState<string | null>(null);

	useEffect(() => {
		if (!open || templates.length !== 0) {
			return;
		}

		async function loadOnce() {
			await loadTemplates();
		}
		loadOnce();
	}, [open, templates.length, loadTemplates]);

	useEffect(() => {
		setCurrentName(selectedTemplate);
		const found = templates.find((tp) => tp.name === selectedTemplate);
		setEditingName(found?.name ?? selectedTemplate ?? "");
		setEditingContent(found?.content ?? "");
	}, [selectedTemplate, templates]);

	const handleSelect = (name: string) => {
		setCurrentName(name);
		const found = templates.find((tp) => tp.name === name);
		setEditingName(found?.name ?? name);
		setEditingContent(found?.content ?? "");
		setSelectedTemplate(name);
		setOpen(false);
	};

	const handleSave = async () => {
		const targetName = editingName.trim();
		if (!(targetName && editTargetId)) {
			return;
		}
		await editTemplate({
			id: editTargetId,
			name: targetName,
			content: editingContent,
		});

		setSelectedTemplate(targetName);
		setCurrentName(targetName);
		setEditOpen(false);
		setOpen(false);
	};

	const handleConfirmDelete = async () => {
		if (!deleteTargetName) {
			return;
		}

		const found = templates.find((tp) => tp.name === deleteTargetName);
		if (!found) {
			setDeleteOpen(false);
			setDeleteTargetName(null);
			return;
		}

		await deleteTemplate(found.id);

		if (currentName === deleteTargetName) {
			setSelectedTemplate(undefined);
			setCurrentName(undefined);
			setTemplateInstructions("");
		}

		setDeleteOpen(false);
		setDeleteTargetName(null);
	};
	const handleEditContentKeyDown = useCallback(
		(event: KeyboardEvent<HTMLTextAreaElement>) => {
			if (event.key !== "Tab") {
				return;
			}
			event.preventDefault();
			const textarea = event.currentTarget;
			const { selectionStart, selectionEnd } = textarea;
			setEditingContent((previous) => {
				const nextValue =
					previous.slice(0, selectionStart) +
					"\t" +
					previous.slice(selectionEnd);
				window.requestAnimationFrame(() => {
					textarea.selectionStart = textarea.selectionEnd = selectionStart + 1;
				});
				return nextValue;
			});
		},
		[setEditingContent]
	);
	const editDisabled = editingName.trim().length === 0;
	const displayLabel =
		currentName ??
		(templates.length > 0
			? t("common.noTemplateSelection")
			: t("common.selectTemplate"));

	return (
		<>
			<Label className="font-medium text-sm" htmlFor={comboboxId}>
				{t("common.templateLabel")}
			</Label>
			{error && (
				<div
					aria-live="assertive"
					className="flex items-center gap-2 rounded-lg border border-destructive bg-destructive/10 px-4 py-3 text-destructive text-xs"
					role="alert"
				>
					<AlertCircle aria-hidden="true" className="h-4 w-4">
						<title>{t("common.errorIconTitle", "Error")}</title>
					</AlertCircle>
					<span>
						{t("common.backendError", "An error occurred:")} {error}
					</span>
				</div>
			)}
			<Popover modal={true} onOpenChange={setOpen} open={open}>
				<PopoverTrigger asChild>
					<Button
						aria-controls={listId}
						aria-expanded={open}
						className={cn(
							"w-full justify-between overflow-hidden hover:text-foreground",
							triggerClasses
						)}
						id={comboboxId}
						onClick={clearError}
						role="combobox"
						type="button"
						variant="outline"
					>
						<span className="min-w-0 flex-1 truncate text-left">
							{displayLabel}
						</span>
						<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
					</Button>
				</PopoverTrigger>
				<PopoverContent
					align="start"
					className={cn(
						"w-[var(--radix-popover-trigger-width)] p-0",
						contentClasses
					)}
				>
					<Command>
						<CommandInput placeholder={t("common.selectTemplate")} />
						<CommandList className="max-h-[240px]" id={listId}>
							<CommandEmpty>{t("common.noTemplateFound")}</CommandEmpty>
							<CommandGroup>
								<CommandItem
									onSelect={() => {
										setSelectedTemplate(undefined);
										setCurrentName(undefined);
										setTemplateInstructions("");
										setOpen(false);
									}}
									value="__clear-template"
								>
									{t("common.noTemplateSelection")}
								</CommandItem>
								<CommandSeparator />
								{templates.map((template) => (
									<CommandItem
										className="group flex items-center justify-between"
										key={template.name}
										onSelect={(value: string) => {
											handleSelect(value);
											setTemplateInstructions(template.content);
										}}
										value={template.name}
									>
										<div className="flex min-w-0 items-center gap-2">
											<CheckIcon
												className={cn(
													"h-4 w-4 flex-shrink-0",
													currentName === template.name
														? "opacity-100"
														: "opacity-0"
												)}
											/>
											<span className="truncate">{template.name}</span>
										</div>
										<div className="ml-2 hidden items-center gap-1 group-hover:flex">
											<Button
												className="h-7 w-7"
												onClick={(e) => {
													e.stopPropagation();
													setEditTargetId(template.id);
													setEditingName(template.name);
													setEditingContent(template.content);
													setEditOpen(true);
												}}
												size="icon"
												title={t("common.editTemplate", "Edit Template")}
												type="button"
												variant="secondary"
											>
												<EditIcon className="h-4 w-4" />
											</Button>

											<Button
												className="h-7 w-7"
												onClick={(e) => {
													e.stopPropagation();
													setDeleteTargetName(template.name);
													setDeleteOpen(true);
												}}
												size="icon"
												title={t("common.delete", "Delete")}
												type="button"
												variant="secondary"
											>
												<Trash className="h-4 w-4" />
											</Button>
										</div>
									</CommandItem>
								))}
							</CommandGroup>
						</CommandList>
					</Command>
				</PopoverContent>
			</Popover>

			<Dialog onOpenChange={setEditOpen} open={editOpen}>
				<DialogContent className="sm:max-w-3xl">
					<DialogHeader>
						<DialogTitle className="text-lg">
							{t("common.editTemplate", "Edit Template")}
						</DialogTitle>
						<DialogDescription className="text-muted-foreground">
							{t(
								"common.editTemplateHelp",
								"Update the template name and content. Your changes will apply immediately after saving."
							)}
						</DialogDescription>
					</DialogHeader>

					<div className="grid gap-4 py-2">
						<div className="grid gap-2">
							<Label htmlFor="template-name">
								{t("common.templateName", "Template Name")}
							</Label>
							<Input
								id="template-name"
								onChange={(e) => setEditingName(e.target.value)}
								placeholder={t(
									"common.templateNamePlaceholder",
									"e.g., Default Instructions"
								)}
								value={editingName}
							/>
							{editingName.trim().length === 0 && (
								<p className="text-muted-foreground text-xs">
									{t("common.nameRequired", "A name is required to save.")}
								</p>
							)}
						</div>

						<div className="grid gap-2">
							<div className="flex items-center justify-between">
								<Label htmlFor="template-content">
									{t("common.templateContent", "Template Content")}
								</Label>
								<span className="text-muted-foreground text-xs">
									{t("common.characters", "Characters")} {editingContent.length}
								</span>
							</div>
							<Textarea
								className="h-56 w-full resize-none rounded-xl border bg-muted/30 font-mono text-sm shadow-inner"
								id="template-content"
								onChange={(e) => setEditingContent(e.target.value)}
								onKeyDown={handleEditContentKeyDown}
								value={editingContent}
							/>
							<p className="text-muted-foreground text-xs">
								{t(
									"common.templateContentHelp",
									"Describe how the assistant should behave. Markdown is supported."
								)}
							</p>
						</div>
					</div>

					<DialogFooter className="gap-2 sm:justify-between">
						<div className="text-muted-foreground text-xs" />
						<div className="flex w-full justify-end gap-2 sm:w-auto">
							<Button
								onClick={() => setEditOpen(false)}
								type="button"
								variant="ghost"
							>
								{t("common.cancel", "Cancel")}
							</Button>
							<Button
								disabled={editDisabled}
								onClick={handleSave}
								type="button"
							>
								{t("common.saveChanges", "Save Changes")}
							</Button>
						</div>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			<Dialog onOpenChange={setDeleteOpen} open={deleteOpen}>
				<DialogContent className="sm:max-w-md">
					<DialogHeader>
						<DialogTitle>
							{t("common.confirmDelete", "Delete this template?")}
						</DialogTitle>
						<DialogDescription>
							{t(
								"common.confirmDeleteHelp",
								"This action cannot be undone. This will permanently remove the template."
							)}
						</DialogDescription>
					</DialogHeader>
					<div className="rounded-md border bg-muted/40 p-3 text-sm">
						{deleteTargetName}
					</div>
					<DialogFooter className="gap-2">
						<Button
							onClick={() => setDeleteOpen(false)}
							type="button"
							variant="ghost"
						>
							{t("common.cancel", "Cancel")}
						</Button>
						<Button
							disabled={!deleteTargetName}
							onClick={handleConfirmDelete}
							type="button"
							variant="destructive"
						>
							{t("common.delete", "Delete")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
};
