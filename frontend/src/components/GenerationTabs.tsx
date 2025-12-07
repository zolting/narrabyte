import type { models } from "@go/models";
import { useTranslation } from "react-i18next";
import { ActivityFeed } from "@/components/ActivityFeed";
import { DocGenerationResultPanel } from "@/components/DocGenerationResultPanel";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import type { ChatMessage, DocGenerationStatus } from "@/stores/docGeneration";
import type { TodoItem, ToolEvent } from "@/types/events";

interface GenerationTabsProps {
	activeTab: "activity" | "review";
	setActiveTab: (tab: "activity" | "review") => void;
	events: ToolEvent[];
	todos: TodoItem[];
	messages: ChatMessage[];
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
	messages,
	status,
	docResult,
	projectId,
	sessionKey,
}: GenerationTabsProps) => {
	const { t } = useTranslation();

	const getGridColumns = () => {
		if (docResult) {
			return "grid-cols-2";
		}
		return "grid-cols-1";
	};

	return (
		<Tabs
			className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
			onValueChange={(value) => setActiveTab(value as "activity" | "review")}
			value={activeTab}
		>
			<TabsList
				className={cn("grid h-auto w-full bg-muted p-1", getGridColumns())}
			>
				<TabsTrigger value="activity">{t("common.recentActivity")}</TabsTrigger>
				{docResult && (
					<TabsTrigger value="review">{t("common.review")}</TabsTrigger>
				)}
			</TabsList>
			<TabsContent
				className="mt-0 flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
				forceMount
				value="activity"
			>
				<ActivityFeed
					events={events}
					messages={messages}
					status={status}
					summary={docResult?.summary ?? null}
					todos={todos}
				/>
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
		</Tabs>
	);
};
