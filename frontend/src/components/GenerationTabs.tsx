import type { models } from "@go/models";
import { useTranslation } from "react-i18next";
import { DocGenerationProgressLog } from "@/components/DocGenerationProgressLog";
import { DocGenerationResultPanel } from "@/components/DocGenerationResultPanel";
import { MarkdownRenderer } from "@/components/MarkdownRenderer";
import { TodoList } from "@/components/TodoList";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import type { DocGenerationStatus } from "@/stores/docGeneration";
import type { DemoEvent, TodoItem } from "@/types/events";

interface GenerationTabsProps {
	activeTab: "activity" | "review" | "summary";
	setActiveTab: (tab: "activity" | "review" | "summary") => void;
	events: DemoEvent[];
	todos: TodoItem[];
	status: DocGenerationStatus;
	docResult: models.DocGenerationResult | null;
	projectId: number;
}

export const GenerationTabs = ({
	activeTab,
	setActiveTab,
	events,
	todos,
	status,
	docResult,
	projectId,
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
				<TabsTrigger
					className={cn(
						"transition-all",
						activeTab === "activity"
							? "!bg-accent !text-accent-foreground shadow-sm"
							: "hover:bg-muted-foreground/10"
					)}
					value="activity"
				>
					{t("common.recentActivity")}
				</TabsTrigger>
				{docResult && (
					<TabsTrigger
						className={cn(
							"transition-all",
							activeTab === "review"
								? "!bg-accent !text-accent-foreground shadow-sm"
								: "hover:bg-muted-foreground/10"
						)}
						value="review"
					>
						{t("common.review")}
					</TabsTrigger>
				)}
				{docResult?.summary && (
					<TabsTrigger
						className={cn(
							"transition-all",
							activeTab === "summary"
								? "!bg-accent !text-accent-foreground shadow-sm"
								: "hover:bg-muted-foreground/10"
						)}
						value="summary"
					>
						{t("common.summary")}
					</TabsTrigger>
				)}
			</TabsList>
			<TabsContent
				className="mt-0 flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
				value="activity"
			>
				<TodoList todos={todos} />
				<DocGenerationProgressLog events={events} status={status} />
			</TabsContent>
			{docResult && (
				<TabsContent
					className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
					value="review"
				>
					<DocGenerationResultPanel projectId={projectId} result={docResult} />
				</TabsContent>
			)}
			{docResult?.summary && (
				<TabsContent
					className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
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
