import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";

interface ComparisonDisplayProps {
	sourceBranch: string | undefined;
	targetBranch: string | undefined;
}

export const ComparisonDisplay = ({
	sourceBranch,
	targetBranch,
}: ComparisonDisplayProps) => {
	const { t } = useTranslation();
	const hasTarget = Boolean((targetBranch ?? "").trim());

	return (
		<div className="flex shrink-0 items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-sm">
			{hasTarget ? (
				<>
					<span className="text-muted-foreground">
						{t("common.comparing")}:
					</span>
					<code className="rounded bg-background px-2 py-1 font-mono text-foreground text-xs">
						{sourceBranch}
					</code>
					<ArrowRight className="h-3 w-3 text-muted-foreground" />
					<code className="rounded bg-background px-2 py-1 font-mono text-foreground text-xs">
						{targetBranch}
					</code>
				</>
			) : (
				<>
					<span className="text-muted-foreground">{t("common.branch")}:</span>
					<code className="rounded bg-background px-2 py-1 font-mono text-foreground text-xs">
						{sourceBranch}
					</code>
				</>
			)}
		</div>
	);
};
