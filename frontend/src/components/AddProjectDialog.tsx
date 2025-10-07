import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import DirectoryPicker from "@/components/DirectoryPicker";
import FilePicker from "@/components/FilePicker";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

type AddProjectDialogProps = {
	open: boolean;
	onClose: () => void;
	onSubmit: (data: {
		name: string;
		docDirectory: string;
		codebaseDirectory: string;
		initFumaDocs: boolean;
		llmInstructions?: string;
	}) => void;
};

export default function AddProjectDialog({
	open,
	onClose,
	onSubmit,
}: AddProjectDialogProps) {
	const { t } = useTranslation();
	const [name, setName] = useState("");
	const [docDirectory, setDocDirectory] = useState("");
	const [codebaseDirectory, setCodebaseDirectory] = useState("");
	const [llmInstructions, setLlmInstructions] = useState("");
	const [initFumaDocs, setInitFumaDocs] = useState<boolean>(false);

	const computedDocDirectory = () => {
		if (!initFumaDocs) {
			return docDirectory;
		}

		const sep = docDirectory.endsWith("/") ? "" : "/";
		return `${docDirectory}${sep}${name}`;
	};

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		onSubmit({
			name,
			docDirectory: computedDocDirectory(),
			codebaseDirectory,
			initFumaDocs,
			llmInstructions: llmInstructions?.trim() ? llmInstructions : undefined,
		});
	};

	useEffect(() => {
		if (open) {
			setName("");
			setDocDirectory("");
			setCodebaseDirectory("");
			setLlmInstructions("");
			setInitFumaDocs(false);
		}
	}, [open]);

	return (
		<Dialog onOpenChange={(isOpen) => !isOpen && onClose()} open={open}>
			<DialogContent className="flex max-h-[90vh] flex-col overflow-hidden sm:max-w-[700px]">
				<DialogHeader className="flex-shrink-0">
					<DialogTitle className="font-semibold text-foreground text-lg">
						{t("projectManager.addProject")}
					</DialogTitle>
				</DialogHeader>

				<form
					className="flex-1 space-y-4 overflow-y-auto overflow-x-hidden px-1"
					onSubmit={handleSubmit}
				>
					<div>
						<label
							className="mb-1 block font-medium text-foreground"
							htmlFor="project-name"
						>
							{t("projectManager.projectName")}
						</label>
						<Input
							className="text-foreground"
							id="project-name"
							onChange={(e) => setName(e.target.value)}
							placeholder={t("projectManager.projectName")}
							required
							value={name}
						/>
					</div>

					<div className="space-y-3">
						<div className="font-medium text-foreground">
							{t("projectManager.chooseNewOrExisitingDoc")}
						</div>
						<div className="flex gap-2">
							<Button
								onClick={() => setInitFumaDocs(false)}
								type="button"
								variant={initFumaDocs ? "outline" : "default"}
							>
								{t("projectManager.existingDocumentationRepository")}
							</Button>
							<Button
								onClick={() => setInitFumaDocs(true)}
								type="button"
								variant={initFumaDocs ? "default" : "outline"}
							>
								{t("projectManager.newDocumentationRepository")}
							</Button>
						</div>

						<div className="space-y-2">
							<label
								className="mb-1 block font-medium text-foreground"
								htmlFor="doc-directory"
							>
								{initFumaDocs
									? t("projectManager.creationPath")
									: t("projectManager.docDirectory")}
							</label>
							<DirectoryPicker
								id="doc-directory"
								onDirectorySelected={setDocDirectory}
							/>
							{docDirectory && (
								<div className="mt-1 text-xs">{computedDocDirectory()}</div>
							)}
						</div>
					</div>

					<div>
						<label
							className="mb-1 block font-medium text-foreground"
							htmlFor="codebase-directory"
						>
							{t("projectManager.codebaseDirectory")}
						</label>
						<DirectoryPicker
							id="codebase-directory"
							onDirectorySelected={setCodebaseDirectory}
						/>
						{codebaseDirectory && (
							<div className="mt-1 text-xs">{codebaseDirectory}</div>
						)}
					</div>

					<div>
						<label
							className="mb-1 block font-medium text-foreground"
							htmlFor="llm-instructions"
						>
							{t("projectManager.llmInstructions")}
						</label>
						<FilePicker
							accept={{
								label: "LLM Prompt",
								extensions: [
									"md",
									"mdx",
									"txt",
									"json",
									"yaml",
									"yml",
									"prompt",
								],
							}}
							id="llm-instructions"
							onFileSelected={setLlmInstructions}
						/>
						{llmInstructions && (
							<div className="mt-1 text-xs">{llmInstructions}</div>
						)}
					</div>
				</form>
				<DialogFooter className="flex-shrink-0 pt-4">
					<Button onClick={onClose} type="button" variant="outline">
						{t("common.cancel")}
					</Button>
					<Button
						disabled={!(name && docDirectory && codebaseDirectory)}
						onClick={handleSubmit}
						type="button"
					>
						{t("home.addProject")}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
