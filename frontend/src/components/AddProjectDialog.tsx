import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import DirectoryPicker from "@/components/DirectoryPicker";
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

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		onSubmit({ name, docDirectory, codebaseDirectory });
	};

	useEffect(() => {
		if (open) {
			setName("");
			setDocDirectory("");
			setCodebaseDirectory("");
		}
	}, [open]);

	return (
		<Dialog onOpenChange={(isOpen) => !isOpen && onClose()} open={open}>
			<DialogContent className="sm:max-w-[520px]">
				<DialogHeader>
					<DialogTitle className="font-semibold text-lg">
						{t("projectManager.addProject")}
					</DialogTitle>
				</DialogHeader>

				<form className="space-y-4" onSubmit={handleSubmit}>
					<div>
						<label
							className="mb-1 block font-medium text-foreground"
							htmlFor="project-name"
						>
							{t("projectManager.projectName")}
						</label>
						<Input
							id="project-name"
							className="text-foreground"
							onChange={(e) => setName(e.target.value)}
							placeholder={t("projectManager.projectName")}
							required
							value={name}
						/>
					</div>

					<div>
						<label className="mb-1 block font-medium" htmlFor="doc-directory">
							{t("projectManager.docDirectory")}
						</label>
						<DirectoryPicker onDirectorySelected={setDocDirectory} />
						{docDirectory && <div className="mt-1 text-xs">{docDirectory}</div>}
					</div>

					<div>
						<label
							className="mb-1 block font-medium"
							htmlFor="codebase-directory"
						>
							{t("projectManager.codebaseDirectory")}
						</label>
						<DirectoryPicker onDirectorySelected={setCodebaseDirectory} />
						{codebaseDirectory && (
							<div className="mt-1 text-xs">{codebaseDirectory}</div>
						)}
					</div>

					<DialogFooter className="pt-2">
						<Button onClick={onClose} type="button" variant="outline">
							{t("common.cancel")}
						</Button>
						<Button
							disabled={!(name && docDirectory && codebaseDirectory)}
							type="submit"
						>
							{t("home.addProject")}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
