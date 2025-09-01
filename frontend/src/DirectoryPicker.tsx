import { useState } from "react";
import { SelectDirectory } from "../wailsjs/go/main/App";
import { Button } from "./components/ui/button";

interface DirectoryPickerProps {
	onDirectorySelected?: (path: string) => void;
}

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
			<div className="flex gap-4 items-center">
				<Button onClick={handleSelectDirectory} disabled={isLoading} size="lg">
					{isLoading ? "Selecting..." : "Select Directory"}
				</Button>
				{selectedPath && (
					<div className="text-sm text-gray-600 truncate max-w-xs">
						Selected: {selectedPath}
					</div>
				)}
			</div>
		</div>
	);
}
