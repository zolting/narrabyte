import type { models } from "@go/models";
import { useTranslation } from "react-i18next";
import { ActivityFeed } from "@/components/ActivityFeed";
import { DocGenerationResultPanel } from "@/components/DocGenerationResultPanel";
import { MarkdownRenderer } from "@/components/MarkdownRenderer";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import type { DocGenerationStatus } from "@/stores/docGeneration";
import type { TodoItem, ToolEvent } from "@/types/events";

interface GenerationTabsProps {
	activeTab: "activity" | "review" | "summary";
	setActiveTab: (tab: "activity" | "review" | "summary") => void;
	events: ToolEvent[];
	todos: TodoItem[];
	status: DocGenerationStatus;
	docResult: models.DocGenerationResult | null;
	projectId: number;
	sessionKey: string | null;
}

export const GenerationTabs = ({
	activeTab,
	setActiveTab,
	events,
	todos,
	status,
	docResult,
	projectId,
	sessionKey,
}: GenerationTabsProps) => {
	const { t } = useTranslation();

	const getGridColumns = () => {
		if (docResult?.summary) {
			return "grid-cols-3";
		}
		if (docResult) {
			return "grid-cols-2";
		}
		return "grid-cols-1";
	};

	return (
		<Tabs
			className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
			onValueChange={(value) =>
				setActiveTab(value as "activity" | "review" | "summary")
			}
			value={activeTab}
		>
			<TabsList
				className={cn("grid h-auto w-full bg-muted p-1", getGridColumns())}
			>
				<TabsTrigger value="activity">{t("common.recentActivity")}</TabsTrigger>
				{docResult && (
					<TabsTrigger value="review">{t("common.review")}</TabsTrigger>
				)}
				{docResult?.summary && (
					<TabsTrigger value="summary">{t("common.summary")}</TabsTrigger>
				)}
			</TabsList>
			<TabsContent
				className="mt-0 flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
				forceMount
				value="activity"
			>
				<ActivityFeed events={events} status={status} todos={todos} />
			</TabsContent>
			{docResult && (
				<TabsContent
					className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
					forceMount
					value="review"
				>
					<DocGenerationResultPanel
						projectId={projectId}
						result={docResult}
						sessionKey={sessionKey}
					/>
				</TabsContent>
			)}
			{docResult?.summary && (
				<TabsContent
					className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
					forceMount
					value="summary"
				>
					<div className="flex h-full flex-col gap-4 overflow-hidden rounded-lg border border-border bg-card p-6">
						<header>
							<h2 className="font-semibold text-foreground text-lg">
								{t("common.summary")}
							</h2>
							<p className="text-muted-foreground text-sm">
								{t("common.branch")}: {docResult.branch}
							</p>
						</header>
						<div className="min-h-0 flex-1 overflow-y-auto">
							<div className="rounded-md border border-border bg-muted/40 p-4 text-foreground/90 leading-relaxed">
								<MarkdownRenderer content={docResult.summary} />
							</div>
						</div>
					</div>
				</TabsContent>
			)}
		</Tabs>
	);
};
