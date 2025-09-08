import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import DirectoryPicker from "@/components/DirectoryPicker";
import { Button } from "@/components/ui/button";
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

export const AddProjectDialog: React.FC<AddProjectDialogProps> = ({
	open,
	onClose,
	onSubmit,
}) => {
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

	if (!open) {
		return null;
	}

	return (
		<div className="dialog-backdrop">
			<div className="dialog">
				<h2 className="mb-4 font-bold text-lg">
					{t("projectManager.addProject")}
				</h2>
				<form className="space-y-4" onSubmit={handleSubmit}>
					<div>
						<label className="mb-1 block font-medium" htmlFor="project-name">
							{t("projectManager.projectName")}
						</label>
						<Input
							id="project-name"
							onChange={(e) => setName(e.target.value)}
							placeholder="Nom du projet"
							required
							value={name}
						/>
					</div>
					<div>
						<label className="mb-1 block font-medium" htmlFor="doc-directory">
							{t("projectManager.docDirectory")}
						</label>
						<DirectoryPicker
							id="doc-directory"
							onDirectorySelected={setDocDirectory}
						/>
						{docDirectory && <div className="mt-1 text-xs">{docDirectory}</div>}
					</div>
					<div>
						<label
							className="mb-1 block font-medium"
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
					<div className="flex justify-end gap-2">
						<Button onClick={onClose} type="button" variant="outline">
							{t("common.cancel")}
						</Button>
						<Button
							disabled={!(name && docDirectory && codebaseDirectory)}
							type="submit"
						>
							{t("home.addProject")}
						</Button>
					</div>
				</form>
			</div>
		</div>
	);
};
