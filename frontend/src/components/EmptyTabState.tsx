import { FileText, History } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTranslation } from "react-i18next";

export type EmptyTabStateProps = {
	onLoadSession: () => void;
	onStartNew: () => void;
};

export function EmptyTabState({ onLoadSession, onStartNew }: EmptyTabStateProps) {
	const { t } = useTranslation();

	return (
		<div className="flex min-h-[400px] flex-col items-center justify-center gap-6 rounded-lg border border-dashed border-border bg-muted/20 p-12">
			<div className="flex flex-col items-center gap-2 text-center">
				<FileText className="h-12 w-12 text-muted-foreground" />
				<h3 className="font-semibold text-foreground text-lg">
					{t("generations.emptyTab.title")}
				</h3>
				<p className="max-w-md text-muted-foreground text-sm">
					{t("generations.emptyTab.description")}
				</p>
			</div>
			<div className="flex items-center gap-3">
				<Button onClick={onLoadSession} variant="outline" type="button">
					<History className="mr-2 h-4 w-4" />
					{t("generations.emptyTab.loadSession")}
				</Button>
				<Button onClick={onStartNew} type="button">
					<FileText className="mr-2 h-4 w-4" />
					{t("generations.emptyTab.startNew")}
				</Button>
			</div>
		</div>
	);
}
