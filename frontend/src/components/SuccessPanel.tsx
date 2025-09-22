import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

interface SuccessPanelProps {
	completedCommitInfo: {
		sourceBranch: string;
		targetBranch: string;
	} | null;
	sourceBranch: string | undefined;
	onStartNewTask: () => void;
}

export const SuccessPanel = ({
	completedCommitInfo,
	sourceBranch,
	onStartNewTask,
}: SuccessPanelProps) => {
	const { t } = useTranslation();

	return (
		<div className="flex flex-col gap-6 rounded-lg border border-green-200 bg-green-50/50 p-6 dark:border-green-800 dark:bg-green-950/30">
			<div className="text-center">
				<h3 className="mb-2 font-semibold text-green-800 text-lg dark:text-green-200">
					{t("common.commitSuccess")}
				</h3>
				<p className="text-green-700 text-sm dark:text-green-300">
					{t("common.commitSuccessDescription")}
				</p>
			</div>
			<div className="text-center">
				<p className="mb-2 text-foreground text-sm">
					{t("common.documentationAvailable")}:
				</p>
				<code className="rounded bg-background px-3 py-2 font-mono text-foreground text-sm shadow-sm">
					docs-{completedCommitInfo?.sourceBranch || sourceBranch}
				</code>
			</div>
			<div className="text-center">
				<Button
					className="gap-2 font-semibold"
					onClick={onStartNewTask}
					type="button"
				>
					{t("common.startNewTask")}
					<ArrowRight className="h-4 w-4" />
				</Button>
			</div>
		</div>
	);
};
