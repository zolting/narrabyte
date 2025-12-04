import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Plus, Trash } from "lucide-react";
import type { KeyboardEvent } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
	AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
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
import { Textarea } from "@/components/ui/textarea";
import { useTemplateStore } from "@/stores/template";

export const Route = createFileRoute("/templates/")({
	component: TemplatesPage,
});

function TemplatesPage() {
	const { t } = useTranslation();

	const templates = useTemplateStore((state) => state.templates);
	const loadTemplates = useTemplateStore((state) => state.loadTemplates);
	const editTemplate = useTemplateStore((state) => state.editTemplate);
	const deleteTemplate = useTemplateStore((state) => state.deleteTemplate);
	const createTemplate = useTemplateStore((state) => state.createTemplate);
	const loading = useTemplateStore((state) => state.loading);
	const error = useTemplateStore((state) => state.error);
	const clearError = useTemplateStore((state) => state.clearError);

	const [selectedId, setSelectedId] = useState<number | null>(null);
	const [pendingSelectionName, setPendingSelectionName] = useState<
		string | null
	>(null);
	const [editingName, setEditingName] = useState("");
	const [editingContent, setEditingContent] = useState("");
	const [newTemplateName, setNewTemplateName] = useState("");
	const [newTemplateContent, setNewTemplateContent] = useState("");
	const [createOpen, setCreateOpen] = useState(false);
	const [isEditing, setIsEditing] = useState(false);

	const hasLoadedRef = useRef(false);

	useEffect(() => {
		if (hasLoadedRef.current) {
			return;
		}
		hasLoadedRef.current = true;
		loadTemplates();
	}, [loadTemplates]);

	useEffect(() => {
		if (!pendingSelectionName) {
			return;
		}

		const match = templates.find(
			(template) => template.name === pendingSelectionName
		);
		if (match) {
			setSelectedId(match.id);
			setPendingSelectionName(null);
		}
	}, [pendingSelectionName, templates]);

	const selectedTemplate = useMemo(
		() => templates.find((template) => template.id === selectedId),
		[selectedId, templates]
	);

	useEffect(() => {
		if (!selectedTemplate) {
			setEditingName("");
			setEditingContent("");
			setIsEditing(false);
			return;
		}
		setEditingName(selectedTemplate.name);
		setEditingContent(selectedTemplate.content);
		setIsEditing(false);
	}, [selectedTemplate]);

	const handleTextareaTab = useCallback(
		(
			event: KeyboardEvent<HTMLTextAreaElement>,
			setter: (updater: (value: string) => string) => void
		) => {
			if (event.key !== "Tab") {
				return;
			}
			event.preventDefault();
			const textarea = event.currentTarget;
			const { selectionStart, selectionEnd } = textarea;
			setter((previous) => {
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
		[]
	);

	const handleSave = async () => {
		const trimmedName = editingName.trim();
		if (!(selectedTemplate && trimmedName)) {
			return;
		}
		clearError();
		await editTemplate({
			id: selectedTemplate.id,
			name: trimmedName,
			content: editingContent,
		});
		if (!useTemplateStore.getState().error) {
			setEditingName(trimmedName);
			setIsEditing(false);
		}
	};

	const handleDelete = async () => {
		if (!selectedTemplate) {
			return;
		}
		clearError();
		await deleteTemplate(selectedTemplate.id);
		if (useTemplateStore.getState().error) {
			return;
		}
		setSelectedId(null);
		setIsEditing(false);
	};

	const handleCreate = async () => {
		const trimmedName = newTemplateName.trim();
		const trimmedContent = newTemplateContent.trim();
		if (!(trimmedName && trimmedContent)) {
			return;
		}
		clearError();
		setPendingSelectionName(trimmedName);
		await createTemplate({
			name: trimmedName,
			content: trimmedContent,
		});
		if (useTemplateStore.getState().error) {
			setPendingSelectionName(null);
			return;
		}
		setNewTemplateName("");
		setNewTemplateContent("");
		setCreateOpen(false);
	};

	const handleCancelEdit = () => {
		if (!selectedTemplate) {
			return;
		}
		setEditingName(selectedTemplate.name);
		setEditingContent(selectedTemplate.content);
		setIsEditing(false);
	};

	return (
		<div className="flex flex-1 flex-col gap-6 bg-background p-4">
			<div className="space-y-2">
				<h1 className="font-semibold text-2xl text-foreground">
					{t("home.templateManagerTitle")}
				</h1>
				<p className="text-muted-foreground text-sm">
					{t("templates.subtitle")}
				</p>
			</div>

			{error && (
				<div
					aria-live="assertive"
					className="flex items-center gap-2 rounded-lg border border-destructive bg-destructive/10 px-4 py-3 text-destructive"
					role="alert"
				>
					<AlertCircle aria-hidden="true" className="h-5 w-5">
						<title>{t("common.errorIconTitle")}</title>
					</AlertCircle>
					<span>
						{t("common.backendError")} {error}
					</span>
				</div>
			)}

			<div className="grid flex-1 gap-6 lg:grid-cols-[280px_1fr]">
				<Card className="flex h-full flex-col">
					<CardHeader className="border-border/60 border-b">
						<CardTitle className="text-base">
							{t("templates.listTitle")}
						</CardTitle>
						<CardDescription>{t("templates.listDescription")}</CardDescription>
					</CardHeader>
					<CardContent className="flex-1 space-y-4 overflow-y-auto pt-4">
						<Button
							className="w-full justify-center"
							onClick={() => setCreateOpen(true)}
							type="button"
							variant="default"
						>
							{t("templates.openCreateButton")}
						</Button>
						<div className="space-y-2">
							{loading && (
								<div className="space-y-2">
									<div className="h-9 animate-pulse rounded-md bg-muted" />
									<div className="h-9 animate-pulse rounded-md bg-muted" />
									<div className="h-9 animate-pulse rounded-md bg-muted" />
								</div>
							)}
							{!loading && templates.length === 0 && (
								<p className="text-muted-foreground text-sm">
									{t("templates.emptyState")}
								</p>
							)}
							{templates.map((template) => (
								<Button
									className="w-full justify-between"
									key={template.id}
									onClick={() => setSelectedId(template.id)}
									type="button"
									variant={template.id === selectedId ? "secondary" : "ghost"}
								>
									<span className="truncate">{template.name}</span>
								</Button>
							))}
						</div>
					</CardContent>
				</Card>

				<Card className="flex h-full flex-col">
					<CardHeader className="border-border/60 border-b">
						<CardTitle className="text-base">
							{selectedTemplate
								? selectedTemplate.name
								: t("templates.detailsTitle")}
						</CardTitle>
						<CardDescription>
							{selectedTemplate
								? t("templates.detailsDescription")
								: t("templates.selectPrompt")}
						</CardDescription>
					</CardHeader>
					<CardContent className="flex flex-1 flex-col gap-4 pt-4">
						{selectedTemplate ? (
							isEditing ? (
								<>
									<div className="grid gap-2">
										<Label htmlFor="template-edit-name">
											{t("common.templateName")}
										</Label>
										<Input
											id="template-edit-name"
											onChange={(event) => setEditingName(event.target.value)}
											placeholder={t("common.templateNamePlaceholder")}
											value={editingName}
										/>
									</div>
									<div className="grid gap-2">
										<div className="flex items-center justify-between">
											<Label htmlFor="template-edit-content">
												{t("common.templateContent")}
											</Label>
											<span className="text-muted-foreground text-xs">
												{t("common.characters")} {editingContent.length}
											</span>
										</div>
										<Textarea
											className="h-64 w-full resize-none rounded-xl border bg-muted/30 font-mono text-sm shadow-inner"
											id="template-edit-content"
											onChange={(event) =>
												setEditingContent(event.target.value)
											}
											onKeyDown={(event) =>
												handleTextareaTab(event, setEditingContent)
											}
											value={editingContent}
										/>
									</div>
									<div className="flex flex-wrap items-center justify-between gap-3 pt-2">
										<AlertDialog>
											<AlertDialogTrigger asChild>
												<Button type="button" variant="destructive">
													<Trash className="mr-2 h-4 w-4" />
													{t("common.delete")}
												</Button>
											</AlertDialogTrigger>
											<AlertDialogContent>
												<AlertDialogHeader>
													<AlertDialogTitle>
														{t("common.confirmDelete")}
													</AlertDialogTitle>
													<AlertDialogDescription>
														{t("common.confirmDeleteHelp")}
													</AlertDialogDescription>
												</AlertDialogHeader>
												<AlertDialogFooter>
													<AlertDialogCancel>
														{t("common.cancel")}
													</AlertDialogCancel>
													<AlertDialogAction onClick={handleDelete}>
														{t("common.delete")}
													</AlertDialogAction>
												</AlertDialogFooter>
											</AlertDialogContent>
										</AlertDialog>
										<div className="flex gap-2">
											<Button
												onClick={handleCancelEdit}
												type="button"
												variant="ghost"
											>
												{t("common.cancel")}
											</Button>
											<Button
												disabled={editingName.trim().length === 0}
												onClick={handleSave}
												type="button"
											>
												{t("common.saveChanges")}
											</Button>
										</div>
									</div>
								</>
							) : (
								<>
									<div className="space-y-2">
										<Label>{t("common.templateName")}</Label>
										<p className="rounded-md border border-border/50 bg-muted/20 px-3 py-2 text-sm">
											{selectedTemplate.name}
										</p>
									</div>
									<div className="space-y-2">
										<Label>{t("common.templateContent")}</Label>
										<div className="h-64 overflow-auto rounded-lg border border-border/60 bg-card/50 p-4 text-muted-foreground text-sm">
											<pre className="whitespace-pre-wrap break-words font-mono">
												{selectedTemplate.content}
											</pre>
										</div>
									</div>
									<div className="flex flex-wrap items-center justify-between gap-3 pt-2">
										<AlertDialog>
											<AlertDialogTrigger asChild>
												<Button type="button" variant="destructive">
													<Trash className="mr-2 h-4 w-4" />
													{t("common.delete")}
												</Button>
											</AlertDialogTrigger>
											<AlertDialogContent>
												<AlertDialogHeader>
													<AlertDialogTitle>
														{t("common.confirmDelete")}
													</AlertDialogTitle>
													<AlertDialogDescription>
														{t("common.confirmDeleteHelp")}
													</AlertDialogDescription>
												</AlertDialogHeader>
												<AlertDialogFooter>
													<AlertDialogCancel>
														{t("common.cancel")}
													</AlertDialogCancel>
													<AlertDialogAction onClick={handleDelete}>
														{t("common.delete")}
													</AlertDialogAction>
												</AlertDialogFooter>
											</AlertDialogContent>
										</AlertDialog>
										<Button onClick={() => setIsEditing(true)} type="button">
											{t("common.editTemplate")}
										</Button>
									</div>
								</>
							)
						) : (
							<div className="flex h-full flex-col items-center justify-center gap-2 text-center">
								<p className="text-muted-foreground text-sm">
									{t("templates.selectPrompt")}
								</p>
							</div>
						)}
					</CardContent>
				</Card>
			</div>

			<Dialog onOpenChange={setCreateOpen} open={createOpen}>
				<DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl lg:max-w-5xl">
					<DialogHeader>
						<DialogTitle>{t("templates.createTitle")}</DialogTitle>
						<DialogDescription>
							{t("templates.createDescription")}
						</DialogDescription>
					</DialogHeader>
					<div className="grid gap-4">
						<div className="grid gap-2">
							<Label htmlFor="template-create-name">
								{t("common.templateName")}
							</Label>
							<Input
								id="template-create-name"
								onChange={(event) => setNewTemplateName(event.target.value)}
								placeholder={t("common.templateNamePlaceholder")}
								value={newTemplateName}
							/>
							{newTemplateName.trim().length === 0 && (
								<p className="text-muted-foreground text-xs">
									{t("common.nameRequired")}
								</p>
							)}
						</div>
						<div className="grid gap-2">
							<div className="flex items-center justify-between">
								<Label htmlFor="template-create-content">
									{t("common.templateContent")}
								</Label>
								<span className="text-muted-foreground text-xs">
									{t("common.characters")} {newTemplateContent.length}
								</span>
							</div>
							<Textarea
								className="h-48 w-full resize-none rounded-xl border bg-muted/30 font-mono text-sm shadow-inner"
								id="template-create-content"
								onChange={(event) => setNewTemplateContent(event.target.value)}
								onKeyDown={(event) =>
									handleTextareaTab(event, setNewTemplateContent)
								}
								value={newTemplateContent}
							/>
							<p className="text-muted-foreground text-xs">
								{t("common.templateContentHelp")}
							</p>
						</div>
					</div>
					<DialogFooter className="gap-2">
						<Button
							onClick={() => {
								setNewTemplateName("");
								setNewTemplateContent("");
								clearError();
								setCreateOpen(false);
							}}
							type="button"
							variant="ghost"
						>
							{t("common.cancel")}
						</Button>
						<Button
							disabled={
								newTemplateName.trim().length === 0 ||
								newTemplateContent.trim().length === 0
							}
							onClick={handleCreate}
							type="button"
						>
							<Plus className="mr-2 h-4 w-4" />
							{t("common.addTemplate")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
