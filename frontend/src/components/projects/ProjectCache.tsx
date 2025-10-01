import { useRouterState } from "@tanstack/react-router";
import type React from "react";
import type { ReactNode } from "react";
import {
	createContext,
	useCallback,
	useContext,
	useMemo,
	useRef,
	useState,
} from "react";
import { ProjectDetailPage } from "./ProjectDetailPage";

const projectDetailRegex = /^\/projects\/[^/]+\/?$/;

interface CacheEntry {
	id: string;
	element: React.ReactElement;
}

interface ProjectCacheContextValue {
	activeProjectId: string | null;
	setActiveProjectId: (id: string | null) => void;
	ensureProject: (id: string) => void;
	getEntries: () => CacheEntry[];
}

const ProjectCacheContext = createContext<ProjectCacheContextValue | undefined>(
	undefined
);

export function useProjectCache() {
	const ctx = useContext(ProjectCacheContext);
	if (!ctx) {
		throw new Error("useProjectCache must be used within ProjectCacheProvider");
	}
	return ctx;
}

export function ProjectCacheProvider({ children }: { children: ReactNode }) {
	const [activeProjectId, setActiveProjectId] = useState<string | null>(null);
	const entriesRef = useRef<Map<string, CacheEntry>>(new Map());
	const [, forceRender] = useState(0);

	const ensureProject = useCallback((id: string) => {
		if (!entriesRef.current.has(id)) {
			entriesRef.current.set(id, {
				id,
				element: <ProjectDetailPage key={id} projectId={id} />,
			});
			forceRender((x) => x + 1);
		}
	}, []);

	const getEntries = useCallback(
		() => Array.from(entriesRef.current.values()),
		[]
	);

	const value = useMemo(
		() => ({ activeProjectId, setActiveProjectId, ensureProject, getEntries }),
		[activeProjectId, ensureProject, getEntries]
	);

	return (
		<ProjectCacheContext.Provider value={value}>
			{children}
		</ProjectCacheContext.Provider>
	);
}

export function ProjectCacheHost() {
	const { activeProjectId, getEntries } = useProjectCache();
	const routerState = useRouterState();
	const pathname = routerState.location.pathname;

	const isProjectDetailView = projectDetailRegex.test(pathname);

	const entries = getEntries();
	if (entries.length === 0) {
		return null;
	}

	const activeEntry = entries.find((e) => e.id === activeProjectId);
	if (!activeEntry) {
		return null;
	}

	return (
		<div
			className="h-full w-full"
			style={{ display: isProjectDetailView ? "block" : "none" }}
		>
			{activeEntry.element}
		</div>
	);
}
