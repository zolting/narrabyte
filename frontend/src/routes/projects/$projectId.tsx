import type { models } from "@go/models";
import { Get } from "@go/services/repoLinkService";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import GenerateDocsDialog from "@/components/GenerateDocsDialog";
import { Button } from "@/components/ui/button";

export const Route = createFileRoute("/projects/$projectId")({
	component: ProjectDetailPage,
});

function ProjectDetailPage() {
	const { t } = useTranslation();
	const { projectId } = Route.useParams();
	const [project, setProject] = useState<models.RepoLink | null>(null);
	const [loading, setLoading] = useState(false);
	const [isGenerateDocsOpen, setIsGenerateDocsOpen] = useState(false);

	useEffect(() => {
		setLoading(true);
		Promise.resolve(Get(Number(projectId)))
			.then((res) => {
				setProject((res as models.RepoLink) ?? null);
			})
			.catch(() => {
				setProject(null);
			})
			.finally(() => {
				setLoading(false);
			});
	}, [projectId]);

	if (loading) {
		return <div className="p-2 text-muted-foreground text-sm">Loading…</div>;
	}

	if (!project) {
		return (
			<div className="p-2 text-muted-foreground text-sm">
				Project not found: {projectId}
			</div>
		);
	}

	return (
		<div className="space-y-4">
			<h1 className="text-center font-semibold text-foreground text-xl shadow-2xl">
				{project.ProjectName}
			</h1>
			<div>
				<Button onClick={() => setIsGenerateDocsOpen(true)}>
					{t("common.generateDocs")}
				</Button>
				<GenerateDocsDialog
					onClose={() => setIsGenerateDocsOpen(false)}
					open={isGenerateDocsOpen}
					project={project}
				/>
			</div>
		</div>
	);
}
