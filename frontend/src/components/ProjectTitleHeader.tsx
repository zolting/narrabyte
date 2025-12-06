import type { models } from "@go/models";
import { Get } from "@go/services/repoLinkService";
import { Link, useLocation, useMatches } from "@tanstack/react-router";
import { List, Settings } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

export function ProjectTitleHeader() {
	const { t } = useTranslation();
	const matches = useMatches();
	const matchWithProject = matches.find(
		(m) => (m.params as Record<string, string>).projectId
	);
	const projectId = (matchWithProject?.params as Record<string, string>)
		?.projectId;
	const [projectName, setProjectName] = useState<string | null>(null);

	useEffect(() => {
		if (!projectId) {
			setProjectName(null);
			return;
		}

		let active = true;
		// Fetch project info
		Promise.resolve(Get(Number(projectId)))
			.then((res: unknown) => {
				if (active && res) {
					const p = res as models.RepoLink;
					setProjectName(p.ProjectName);
				}
			})
			.catch(() => {
				if (active) {
					setProjectName(null);
				}
			});

		return () => {
			active = false;
		};
	}, [projectId]);

	const location = useLocation();

	if (!projectName) {
		return null;
	}

	return (
		<div className="flex items-center gap-4">
			<div className="flex items-center gap-2 font-medium text-lg">
				<span className="text-muted-foreground">/</span>
				<span>{projectName}</span>
			</div>
			<div className="my-auto h-6 w-px bg-border" />
			<div className="flex items-center gap-2">
				<Button
					asChild
					size="sm"
					variant={
						location.pathname.endsWith("/generations") ? "default" : "secondary"
					}
				>
					<Link params={{ projectId }} to="/projects/$projectId/generations">
						<List className="mr-2 h-4 w-4" />
						{t("sidebar.ongoingGenerations")}
					</Link>
				</Button>
				<Button
					asChild
					size="sm"
					variant={
						location.pathname.endsWith("/settings") ? "default" : "secondary"
					}
				>
					<Link params={{ projectId }} to="/projects/$projectId/settings">
						<Settings className="mr-2 h-4 w-4" />
						{t("common.settings")}
					</Link>
				</Button>
			</div>
		</div>
	);
}
