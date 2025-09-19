import { SelectDirectory } from "@go/main/App";
import { useState } from "react";
import { useTranslation } from "react-i18next";
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
	const [isLoading, setIsLoading] = useState(false);

	const handleSelectDirectory = async () => {
		setIsLoading(true);
		try {
			const path = await SelectDirectory();
			if (path) {
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
			</div>
		</div>
	);
}
