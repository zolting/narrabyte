import { useState } from "react";
import { useTranslation } from "react-i18next";
import { SelectDirectory } from "../../wailsjs/go/main/App";
import { Button } from "./ui/button";

type DirectoryPickerProps = {
	onDirectorySelected?: (path: string) => void;
	id?: string;
};

export default function DirectoryPicker({
	onDirectorySelected,
	id,
}: DirectoryPickerProps) {
	const { t } = useTranslation();
	const [selectedPath, setSelectedPath] = useState<string>("");
	const [isLoading, setIsLoading] = useState(false);

	const handleSelectDirectory = async () => {
		setIsLoading(true);
		try {
			const path = await SelectDirectory();
			if (path) {
				setSelectedPath(path);
				onDirectorySelected?.(path);
			}
		} catch {
			// Error handled silently - directory selection failed
		} finally {
			setIsLoading(false);
		}
	};

	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-4">
				<Button
					disabled={isLoading}
					id={id}
					onClick={handleSelectDirectory}
					size="lg"
				>
					{isLoading ? t("common.selecting") : t("common.selectDirectory")}
				</Button>
				{selectedPath && (
					<div className="max-w-xs truncate text-gray-600 text-sm">
						{t("common.selected")}: {selectedPath}
					</div>
				)}
			</div>
		</div>
	);
}
