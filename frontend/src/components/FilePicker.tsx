import { SelectFile, SelectFileFiltered } from "@go/main/App";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "./ui/button";

type FilePickerProps = {
	id?: string;
	onFileSelected?: (path: string) => void;
	accept?: { label: string; extensions: string[] };
	disabled?: boolean;
	className?: string;
};

export default function FilePicker({
	id,
	onFileSelected,
	accept,
	disabled,
	className,
}: FilePickerProps) {
	const { t } = useTranslation();
	const [isLoading, setIsLoading] = useState(false);

	const pick = async () => {
		if (disabled || isLoading) {
			return;
		}
		setIsLoading(true);
		try {
			let path: string | undefined;
			if (accept?.extensions.length) {
				path = await SelectFileFiltered(
					accept.label || "Files",
					accept.extensions
				);
			} else {
				path = await SelectFile();
			}
			if (path) {
				onFileSelected?.(path);
			}
		} catch {
			// Error handled silently - file selection failed
		} finally {
			setIsLoading(false);
		}
	};

	return (
		<div className={className} id={id}>
			<Button disabled={disabled || isLoading} onClick={pick} type="button">
				{isLoading ? t("common.selecting") : t("common.selectFileAction")}
			</Button>
		</div>
	);
}
