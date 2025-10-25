import { CheckIcon, ChevronsUpDownIcon, EditIcon, Trash, AlertCircle } from "lucide-react";
import { useEffect, useState } from "react";
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
	DialogTrigger,
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

export const TemplateSelector = ({
									 setTemplateInstructions,
								 }: TemplateSelectorProps) => {
	const { t } = useTranslation();

	const [selectedTemplate, setSelectedTemplate] = useState<string | undefined>(
		undefined
	);
	const createTemplate = useTemplateStore((state) => state.createTemplate);
	const editTemplate = useTemplateStore((state) => state.editTemplate);
	const deleteTemplate = useTemplateStore((state) => state.deleteTemplate);
	const loadTemplates = useTemplateStore((state) => state.loadTemplates);
	const templates = useTemplateStore((state) => state.templates);

	const error = useTemplateStore((state) => state.error);
	const loading = useTemplateStore((state) => state.loading);
	const clearError = useTemplateStore((state) => state.clearError);

	const [open, setOpen] = useState(false);
	const [editOpen, setEditOpen] = useState(false);
	const [addOpen, setAddOpen] = useState(false);
	const [deleteOpen, setDeleteOpen] = useState(false);

	const [currentName, setCurrentName] = useState<string | undefined>(
		selectedTemplate
	);

	const [editingName, setEditingName] = useState("");
	const [editingContent, setEditingContent] = useState("");
	const [newName, setNewName] = useState("");
	const [newContent, setNewContent] = useState("");
	const [editTargetId, setEditTargetId] = useState<number | null>(null);
	const [deleteTargetName, setDeleteTargetName] = useState<string | null>(null);

	useEffect(() => {
		if (!open || templates.length !== 0) return;

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

	const handleAddSave = async () => {
		const nameTrim = newName.trim();
		if (!nameTrim) {
			return;
		}

		await createTemplate({
			name: nameTrim,
			content: newContent,
		});

		setSelectedTemplate(nameTrim);
		setCurrentName(nameTrim);
		setTemplateInstructions(newContent);
		setAddOpen(false);
		setOpen(false);
		setNewName("");
		setNewContent("");
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
			setSelectedTemplate(templates.length ? templates[0]?.name : "");
			setCurrentName(undefined);
			setTemplateInstructions("");
		}

		setDeleteOpen(false);
		setDeleteTargetName(null);
	};

	const addDisabled =
		newName.trim().length === 0 || newContent.trim().length === 0;
	const editDisabled = editingName.trim().length === 0;

	return (
		<div className="shrink-0 space-y-2">
			{error && (
				<div
					role="alert"
					aria-live="assertive"
					className="mb-2 flex items-center gap-2 rounded-lg border border-destructive bg-destructive/10 px-4 py-3 text-destructive"
				>
					<AlertCircle className="h-5 w-5" aria-hidden="true">
						<title>{t("common.errorIconTitle", "Error")}</title>
					</AlertCircle>
					<span>
            {t("common.backendError", "An error occurred:")} {error}
          </span>
				</div>
			)}
		<div className="shrink-0 space-y-2">
			<Label className="font-medium text-sm" htmlFor="provider-select">
				{t("common.templateLabel")}
			</Label>

			<Popover onOpenChange={setOpen} open={open}>
				<PopoverTrigger asChild>
					<Button
						aria-expanded={open}
						className="w-full justify-between"
						type="button"
						variant="outline"
						onClick={clearError}
					>
						{currentName ?? t("common.selectTemplate")}
						<ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
					</Button>
				</PopoverTrigger>

				<PopoverContent className="w-[var(--radix-popover-trigger-width)] max-w-none p-2">
					<div className="rounded-2xl border bg-background shadow-sm">
						<Command className="rounded-2xl">
							<CommandInput placeholder={t("common.selectTemplate")} />
							<CommandList>
								<CommandEmpty>{t("common.noTemplateFound")}</CommandEmpty>
								<CommandGroup>
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

						<div className="border-t" />

						<div className="flex flex-col gap-2 p-3 sm:flex-row sm:items-center sm:justify-end">
							<Button
								onClick={() => setOpen(false)}
								type="button"
								variant="ghost"
							>
								{t("common.close", "Close")}
							</Button>

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
													{t(
														"common.nameRequired",
														"A name is required to save."
													)}
												</p>
											)}
										</div>

										<div className="grid gap-2">
											<div className="flex items-center justify-between">
												<Label htmlFor="template-content">
													{t("common.templateContent", "Template Content")}
												</Label>
												<span className="text-muted-foreground text-xs">
                          {t("common.characters", "Characters")}{" "}
													{editingContent.length}
                        </span>
											</div>
											<Textarea
												className="h-56 w-full resize-none rounded-xl border bg-muted/30 font-mono text-sm shadow-inner"
												id="template-content"
												onChange={(e) => setEditingContent(e.target.value)}
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
								<DialogTrigger asChild>
									<span className="hidden" />
								</DialogTrigger>
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

							<Dialog onOpenChange={setAddOpen} open={addOpen}>
								<DialogTrigger asChild>
									<span className="hidden" />
								</DialogTrigger>
								<DialogContent className="sm:max-w-3xl">
									<DialogHeader>
										<DialogTitle className="text-lg">
											{t("common.addTemplate", "Add Template")}
										</DialogTitle>
										<DialogDescription className="text-muted-foreground">
											{t(
												"common.addTemplateHelp",
												"Create a new template by providing a name and content. You can edit it later."
											)}
										</DialogDescription>
									</DialogHeader>

									<div className="grid gap-4 py-2">
										<div className="grid gap-2">
											<Label htmlFor="new-template-name">
												{t("common.templateName", "Template Name")}
											</Label>
											<Input
												id="new-template-name"
												onChange={(e) => setNewName(e.target.value)}
												placeholder={t(
													"common.templateNamePlaceholder",
													"e.g., Default Instructions"
												)}
												value={newName}
											/>
											{newName.trim().length === 0 && (
												<p className="text-muted-foreground text-xs">
													{t(
														"common.nameRequired",
														"A name is required to save."
													)}
												</p>
											)}
										</div>

										<div className="grid gap-2">
											<div className="flex items-center justify-between">
												<Label htmlFor="new-template-content">
													{t("common.templateContent", "Template Content")}
												</Label>
												<span className="text-muted-foreground text-xs">
                          {t("common.characters", "Characters")}{" "}
													{newContent.length}
                        </span>
											</div>
											<Textarea
												className="h-56 w-full resize-none rounded-xl border bg-muted/30 font-mono text-sm shadow-inner"
												id="new-template-content"
												onChange={(e) => setNewContent(e.target.value)}
												value={newContent}
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
												onClick={() => {
													setAddOpen(false);
													setNewName("");
													setNewContent("");
												}}
												type="button"
												variant="ghost"
											>
												{t("common.cancel", "Cancel")}
											</Button>
											<Button
												disabled={addDisabled}
												onClick={handleAddSave}
												type="button"
											>
												{t("common.addTemplate", "Add Template")}
											</Button>
										</div>
									</DialogFooter>
								</DialogContent>
							</Dialog>

							<Button onClick={() => setAddOpen(true)} type="button">
								{t("common.addTemplate", "Add Template")}
							</Button>
						</div>
					</div>
				</PopoverContent>
			</Popover>
		</div>
	</div>
	);
};