import { useState } from "react";
import { SelectDirectory } from "../../wailsjs/go/main/App";
import { Button } from "./ui/button";

type DirectoryPickerProps = {
	onDirectorySelected?: (path: string) => void;
};

export default function DirectoryPicker({
	onDirectorySelected,
}: DirectoryPickerProps) {
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
		} catch (error) {
			console.error("Error selecting directory:", error);
		} finally {
			setIsLoading(false);
		}
	};

	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-4">
				<Button disabled={isLoading} onClick={handleSelectDirectory} size="lg">
					{isLoading ? "Selecting..." : "Select Directory"}
				</Button>
				{selectedPath && (
					<div className="max-w-xs truncate text-gray-600 text-sm">
						Selected: {selectedPath}
					</div>
				)}
			</div>
		</div>
	);
}
