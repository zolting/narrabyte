import { createFileRoute } from "@tanstack/react-router";
import { useEffect } from "react";
import { useProjectCache } from "@/components/projects/ProjectCache";

export const Route = createFileRoute("/projects/$projectId/")({
	component: ProjectRouteWrapper,
});

function ProjectRouteWrapper() {
	const { projectId } = Route.useParams();
	const { ensureProject, setActiveProjectId } = useProjectCache();

	useEffect(() => {
		ensureProject(projectId);
		setActiveProjectId(projectId);
	}, [projectId, ensureProject, setActiveProjectId]);
}
