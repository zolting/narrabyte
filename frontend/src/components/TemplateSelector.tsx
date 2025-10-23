// TypeScript

import type { models } from "@go/models";
import { CheckIcon } from "lucide-react";
import React, { useEffect, useState } from "react";
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
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface TemplateSelectorProps {
	templates: models.Template[];
	selectedTemplate: string | undefined;
	setSelectedTemplate: (template: string) => void;
	onUpdateTemplate?: (name: string, content: string) => void;
}

export const TemplateSelector = ({
	templates,
	selectedTemplate,
	setSelectedTemplate,
	onUpdateTemplate,
}: TemplateSelectorProps) => {
	const { t } = useTranslation();
	const [open, setOpen] = useState(false);
	const [currentName, setCurrentName] = useState<string | undefined>(
		selectedTemplate
	);
	const [editingContent, setEditingContent] = useState("");

	useEffect(() => {
		setCurrentName(selectedTemplate);
		const found = templates.find((tp) => tp.name === selectedTemplate);
		setEditingContent(found?.content ?? "");
	}, [selectedTemplate, templates]);

	const handleSelect = (name: string) => {
		setCurrentName(name);
		const found = templates.find((tp) => tp.name === name);
		setEditingContent(found?.content ?? "");
		setSelectedTemplate(name);
	};

	const handleSave = () => {
		if (!currentName) return;
		onUpdateTemplate?.(currentName, editingContent);
		setOpen(false);
	};

	return (
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
					>
						{currentName ?? t("common.selectTemplate")}
					</Button>
				</PopoverTrigger>
				<PopoverContent className="w-[320px]">
					<Command>
						<CommandInput placeholder={t("common.selectTemplate")} />
						<CommandList>
							<CommandEmpty>{t("common.noTemplateFound")}</CommandEmpty>
							<CommandGroup>
								{templates.map((template) => (
									<CommandItem
										key={template.name}
										onSelect={(value: string) => {
											handleSelect(value);
											// keep popover open so user can edit after selecting
										}}
										value={template.name}
									>
										<CheckIcon
											className={cn(
												"mr-2 h-4 w-4",
												currentName === template.name
													? "opacity-100"
													: "opacity-0"
											)}
										/>
										{template.name}
									</CommandItem>
								))}
							</CommandGroup>
						</CommandList>
					</Command>

					<div className="mt-3 space-y-2">
						<Label className="font-medium text-sm" htmlFor="template-content">
							{t("common.editTemplateLabel", "Edit template")}
						</Label>
						<Textarea
							id="template-content"
							onChange={(e) => setEditingContent(e.target.value)}
							placeholder={t(
								"common.editTemplatePlaceholder",
								"Template content"
							)}
							rows={6}
							value={editingContent}
						/>
						<div className="flex justify-end gap-2">
							<Button
								onClick={() => setOpen(false)}
								type="button"
								variant="ghost"
							>
								{t("common.close", "Close")}
							</Button>
							<Button onClick={handleSave} type="button">
								{t("common.save", "Save")}
							</Button>
						</div>
					</div>
				</PopoverContent>
			</Popover>
		</div>
	);
};
