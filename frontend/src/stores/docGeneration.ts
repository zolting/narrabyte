import type { models } from "@go/models";
import {
	CommitDocs,
	GenerateDocs,
	GenerateDocsFromBranch,
	LoadGenerationSession,
	MergeDocsIntoSource,
	RefineDocs,
	StopStream,
} from "@go/services/ClientService";
import { DeleteBranchByPath } from "@go/services/GitService";
import { Get as GetRepoLink } from "@go/services/repoLinkService";
import i18n from "i18next";
import { parseDiff } from "react-diff-view";
import { create } from "zustand";
import {
	type TodoItem,
	type ToolEvent,
	todoEventSchema,
	toolEventSchema,
} from "@/types/events";
import { EventsOn } from "../../wailsjs/runtime";

export type DocGenerationStatus =
	| "idle"
	| "running"
	| "success"
	| "error"
	| "committing"
	| "canceled";

type StartArgs = {
	projectId: number;
	projectName: string;
	sourceBranch: string;
	targetBranch: string;
	modelKey: string;
	userInstructions: string;
};

type CommitArgs = {
	projectId: number;
	branch: string;
	files: string[];
};

type ProjectKey = string;
export type SessionKey = string;
type TabId = string;

type SessionMeta = {
	projectId: number;
	projectName: string;
	sourceBranch: string;
	targetBranch: string;
	status: DocGenerationStatus;
};

type CompletedCommitInfo = {
	sourceBranch: string;
	targetBranch: string;
	wasMerge?: boolean;
};

type ChatMessage = {
	id: string;
	role: "user" | "assistant";
	content: string;
	status?: "pending" | "sent" | "error";
	createdAt: Date;
};

type DocsBranchConflict = {
	existingDocsBranch: string;
	proposedDocsBranch: string;
	mode: "diff" | "single";
	isInProgress?: boolean; // true if conflict is due to ongoing generation, false/undefined if branch just exists
};

type DocGenerationData = {
	projectId: number;
	projectName: string;
	sessionKey: SessionKey;
	events: ToolEvent[];
	todos: TodoItem[];
	status: DocGenerationStatus;
	result: models.DocGenerationResult | null;
	error: string | null;
	cancellationRequested: boolean;
	activeTab: "activity" | "review" | "summary";
	commitCompleted: boolean;
	completedCommitInfo: CompletedCommitInfo | null;
	sourceBranch: string | null;
	targetBranch: string | null;
	chatOpen: boolean;
	messages: ChatMessage[];
	initialDiffSignatures: Record<string, string> | null;
	changedSinceInitial: string[];
	docsInCodeRepo: boolean;
	docsBranch: string | null;
	mergeInProgress: boolean;
	conflict?: DocsBranchConflict | null;
};

type State = {
	// Session data keyed by sessionKey instead of projectKey
	docStates: Record<SessionKey, DocGenerationData>;
	// Mapping of projects to tabs to sessions
	tabSessions: Record<ProjectKey, Record<TabId, SessionKey | null>>;
	// Metadata for all sessions (running, completed, background)
	sessionMeta: Record<SessionKey, SessionMeta>;
	// Active session per project (for backward compatibility)
	activeSession: Record<ProjectKey, SessionKey | null>;
	// Tab management actions
	createTabSession: (
		projectId: number,
		tabId: TabId,
		sessionKey: SessionKey | null
	) => void;
	removeTabSession: (projectId: number, tabId: TabId) => void;
	getSessionForTab: (projectId: number, tabId: TabId) => SessionKey | null;

	// Generation actions (now accept optional tabId)
	start: (args: StartArgs & { tabId?: TabId }) => Promise<void>;
	startFromBranch?: (args: StartArgs & { tabId?: TabId }) => Promise<void>;
	reset: (projectId: number | string, sessionKey?: SessionKey) => void;
	commit: (args: CommitArgs & { sessionKey?: SessionKey }) => Promise<void>;
	cancel: (
		projectId: number | string,
		sessionKey?: SessionKey
	) => Promise<void>;
	setActiveTab: (
		sessionKey: SessionKey,
		tab: "activity" | "review" | "summary"
	) => void;
	setCommitCompleted: (sessionKey: SessionKey, completed: boolean) => void;
	setCompletedCommitInfo: (
		sessionKey: SessionKey,
		info: CompletedCommitInfo | null
	) => void;
	toggleChat: (sessionKey: SessionKey, open?: boolean) => void;
	refine: (args: {
		projectId: number;
		branch: string;
		instruction: string;
		sessionKey?: SessionKey;
	}) => Promise<void>;
	mergeDocs: (args: {
		projectId: number;
		branch: string;
		sessionKey?: SessionKey;
	}) => Promise<void>;
	restoreSession: (
		projectId: number,
		sourceBranch: string,
		targetBranch: string,
		tabId?: TabId
	) => Promise<boolean>;
	setActiveSession: (
		projectId: number | string,
		sessionKey: SessionKey | null
	) => void;
	clearSessionMeta: (projectId: number, sourceBranch: string) => void;
	resolveDocsBranchConflictByDelete: (args: {
		projectId: number;
		projectName: string;
		sourceBranch: string;
		mode: "diff" | "single";
		targetBranch?: string;
		modelKey: string;
		userInstructions: string;
		tabId?: TabId;
		sessionKey?: SessionKey;
	}) => Promise<void>;
	resolveDocsBranchConflictByRename: (args: {
		projectId: number;
		sourceBranch: string;
		newDocsBranch: string;
		mode: "diff" | "single";
		targetBranch?: string;
		modelKey: string;
		userInstructions: string;
		tabId?: TabId;
		sessionKey?: SessionKey;
	}) => Promise<void>;
	clearConflict: (sessionKey: SessionKey) => void;
};

const EMPTY_DOC_STATE: DocGenerationData = {
	projectId: 0,
	projectName: "",
	sessionKey: "",
	events: [],
	todos: [],
	status: "idle",
	result: null,
	error: null,
	cancellationRequested: false,
	activeTab: "activity",
	commitCompleted: false,
	completedCommitInfo: null,
	sourceBranch: null,
	targetBranch: null,
	chatOpen: false,
	messages: [],
	initialDiffSignatures: null,
	changedSinceInitial: [],
	docsInCodeRepo: false,
	docsBranch: null,
	mergeInProgress: false,
};

const toKey = (projectId: number | string): ProjectKey => String(projectId);
const normalizeTabId = (tabId?: TabId | null): string | null => {
	const normalized = (tabId ?? "").trim();
	return normalized === "" ? null : normalized;
};

export const createSessionKey = (
	projectId: number,
	sourceBranch: string | null | undefined,
	tabId?: TabId | null
): SessionKey => {
	const normalizedBranch = (sourceBranch ?? "").trim();
	const baseKey = `${projectId}:${normalizedBranch}`;
	const normalizedTab = normalizeTabId(tabId);
	return normalizedTab ? `${baseKey}:${normalizedTab}` : baseKey;
};

const backendSessionBindings = new Map<SessionKey, SessionKey>();

const bindBackendSession = (
	baseSessionKey: SessionKey,
	sessionKey: SessionKey
) => {
	backendSessionBindings.set(baseSessionKey, sessionKey);
};

const unbindBackendSession = (
	baseSessionKey: SessionKey,
	sessionKey?: SessionKey
) => {
	if (sessionKey && backendSessionBindings.get(baseSessionKey) !== sessionKey) {
		return;
	}
	backendSessionBindings.delete(baseSessionKey);
};

const isEventForSession = (
	incomingKey: string | undefined,
	sessionKey: SessionKey,
	baseSessionKey: SessionKey
) => {
	if (!incomingKey) {
		return true;
	}
	if (incomingKey === sessionKey) {
		return true;
	}
	if (incomingKey !== baseSessionKey) {
		return false;
	}
	const boundSession = backendSessionBindings.get(baseSessionKey);
	if (boundSession) {
		return boundSession === sessionKey;
	}
	return sessionKey === baseSessionKey;
};

const messageFromError = (error: unknown) => {
	if (error instanceof Error) {
		return error.message;
	}
	if (typeof error === "string") {
		return error;
	}
	return "An unknown error occurred while generating documentation.";
};

const mapErrorCodeToMessage = (errorMessage: string): string => {
	const trimmed = errorMessage.trim();

	// Map conflict errors to user-friendly messages
	if (
		trimmed.startsWith("ERR_DOCS_BRANCH_EXISTS_SUGGEST:") ||
		trimmed.startsWith("ERR_DOCS_GENERATION_IN_PROGRESS_SUGGEST:")
	) {
		return i18n.t("common.docsBranchConflict");
	}

	if (
		trimmed.startsWith("ERR_DOCS_BRANCH_EXISTS:") ||
		trimmed.startsWith("ERR_DOCS_GENERATION_IN_PROGRESS:")
	) {
		return i18n.t("common.docsBranchExists");
	}

	// Check for specific error codes
	if (trimmed === "ERR_UNCOMMITTED_CHANGES_ON_SOURCE_BRANCH") {
		return i18n.t("common.mergeDisabledUncommittedChanges");
	}
	return errorMessage;
};

const extractExistingDocsBranch = (errorMessage: string): string | null => {
	const idx = errorMessage.indexOf("ERR_DOCS_BRANCH_EXISTS:");
	if (idx === -1) {
		return null;
	}
	const after = errorMessage
		.slice(idx + "ERR_DOCS_BRANCH_EXISTS:".length)
		.trim();
	return after || null;
};

// Extract branch conflict information from error messages with suggestions
// Handles formats like:
// - ERR_DOCS_BRANCH_EXISTS_SUGGEST:docs/feature:docs/feature-2
// - ERR_DOCS_GENERATION_IN_PROGRESS_SUGGEST:docs/feature:docs/feature-2
const extractBranchConflictSuggestion = (
	errorMessage: string
): { existing: string; proposed: string; isInProgress: boolean } | null => {
	const inProgressPrefix = "ERR_DOCS_GENERATION_IN_PROGRESS_SUGGEST:";
	const existsPrefix = "ERR_DOCS_BRANCH_EXISTS_SUGGEST:";

	let isInProgress = false;
	let remainder = "";

	if (errorMessage.startsWith(inProgressPrefix)) {
		isInProgress = true;
		remainder = errorMessage.slice(inProgressPrefix.length);
	} else if (errorMessage.startsWith(existsPrefix)) {
		isInProgress = false;
		remainder = errorMessage.slice(existsPrefix.length);
	} else {
		return null;
	}

	// Parse "existing:proposed" format
	const colonIndex = remainder.indexOf(":");
	if (colonIndex === -1) {
		return null;
	}

	const existing = remainder.slice(0, colonIndex).trim();
	const proposed = remainder.slice(colonIndex + 1).trim();

	if (!(existing && proposed)) {
		return null;
	}

	return { existing, proposed, isInProgress };
};

const createLocalEvent = (
	type: ToolEvent["type"],
	message: string
): ToolEvent => ({
	id:
		typeof crypto !== "undefined" && "randomUUID" in crypto
			? crypto.randomUUID()
			: Math.random().toString(36).slice(2),
	message,
	type,
	timestamp: new Date(),
});

const STARTS_WITH_A_SLASH_REGEX = /^a\//;
const STARTS_WITH_B_SLASH_REGEX = /^b\//;

const normalizeDiffPath = (path?: string | null) => {
	if (!path) {
		return "";
	}
	return path
		.replace(STARTS_WITH_A_SLASH_REGEX, "")
		.replace(STARTS_WITH_B_SLASH_REGEX, "");
};

const computeDiffSignatures = (diffText: string | null | undefined) => {
	if (!diffText?.trim()) {
		return {};
	}
	try {
		const files = parseDiff(diffText);
		const signatures: Record<string, string> = {};
		for (const file of files) {
			const key = normalizeDiffPath(
				file.newPath && file.newPath !== "/dev/null"
					? file.newPath
					: file.oldPath
			);
			signatures[key] = JSON.stringify(
				(file.hunks ?? []).map((hunk) => ({
					content: hunk.content,
					changes: hunk.changes.map((change) => ({
						type: change.type,
						content: change.content,
					})),
				}))
			);
		}
		return signatures;
	} catch (error) {
		console.error("Failed to parse diff when computing signatures", error);
		return {};
	}
};

type SubscriptionMap = {
	tool?: () => void;
	done?: () => void;
	todo?: () => void;
};

// Subscriptions are now per-session instead of per-project
const subscriptions = new Map<SessionKey, SubscriptionMap>();

const clearSubscriptions = (key: SessionKey) => {
	const entry = subscriptions.get(key);
	if (!entry) {
		return;
	}
	entry.tool?.();
	entry.done?.();
	entry.todo?.();
	subscriptions.delete(key);
};

export const useDocGenerationStore = create<State>((set, get, _api) => {
	// Updated to use SessionKey instead of ProjectKey
	const setDocState = (
		sessionKey: SessionKey,
		partial:
			| Partial<DocGenerationData>
			| ((prev: DocGenerationData) => DocGenerationData)
	) => {
		set((state) => {
			const previous = state.docStates[sessionKey] ?? EMPTY_DOC_STATE;
			const next =
				typeof partial === "function"
					? partial(previous)
					: { ...previous, ...partial };
			return {
				docStates: {
					...state.docStates,
					[sessionKey]: next,
				},
			};
		});
	};

	const setActiveSessionKey = (
		projectKey: ProjectKey,
		sessionKey: SessionKey | null
	) => {
		set((state) => ({
			activeSession: {
				...state.activeSession,
				[projectKey]: sessionKey,
			},
		}));
	};

	const updateSessionMeta = (
		sessionKey: SessionKey,
		updater:
			| Partial<SessionMeta>
			| ((prev: SessionMeta | undefined) => SessionMeta)
	) => {
		set((state) => {
			const previous = state.sessionMeta[sessionKey];
			let next: SessionMeta | undefined;
			if (typeof updater === "function") {
				next = updater(previous);
			} else {
				next = { ...previous, ...updater } as SessionMeta;
			}
			if (!next) {
				return state;
			}
			return {
				sessionMeta: {
					...state.sessionMeta,
					[sessionKey]: next,
				},
			};
		});
	};

	const removeSessionMeta = (sessionKey: SessionKey) => {
		set((state) => {
			if (!(sessionKey in state.sessionMeta)) {
				return state;
			}
			const { [sessionKey]: _, ...rest } = state.sessionMeta;
			return {
				sessionMeta: rest,
			};
		});
	};

	// Tab management helpers
	const setTabSession = (
		projectId: number,
		tabId: TabId,
		sessionKey: SessionKey | null
	) => {
		const projectKey = toKey(projectId);
		set((state) => ({
			tabSessions: {
				...state.tabSessions,
				[projectKey]: {
					...(state.tabSessions[projectKey] ?? {}),
					[tabId]: sessionKey,
				},
			},
		}));
	};

	const getTabSession = (
		projectId: number,
		tabId: TabId
	): SessionKey | null => {
		const projectKey = toKey(projectId);
		const state = get();
		return state.tabSessions[projectKey]?.[tabId] ?? null;
	};

	const removeTab = (projectId: number, tabId: TabId) => {
		const projectKey = toKey(projectId);
		set((state) => {
			const projectTabs = state.tabSessions[projectKey];
			if (!(projectTabs && tabId in projectTabs)) {
				return state;
			}
			const { [tabId]: _, ...remainingTabs } = projectTabs;
			return {
				tabSessions: {
					...state.tabSessions,
					[projectKey]: remainingTabs,
				},
			};
		});
	};

	return {
		docStates: {},
		tabSessions: {},
		sessionMeta: {},
		activeSession: {},

		// Tab management actions
		createTabSession: (
			projectId: number,
			tabId: TabId,
			sessionKey: SessionKey | null
		) => {
			setTabSession(projectId, tabId, sessionKey);
		},

		removeTabSession: (projectId: number, tabId: TabId) => {
			removeTab(projectId, tabId);
		},

		getSessionForTab: (projectId: number, tabId: TabId): SessionKey | null => {
			return getTabSession(projectId, tabId);
		},

		start: async ({
			projectId,
			projectName,
			sourceBranch,
			targetBranch,
			modelKey,
			userInstructions,
			tabId,
		}: StartArgs & { tabId?: TabId }) => {
			const projectKey = toKey(projectId);
			const baseSessionKey = createSessionKey(projectId, sourceBranch);
			const sessionKey = tabId
				? createSessionKey(projectId, sourceBranch, tabId)
				: baseSessionKey;
			const currentState = get().docStates[sessionKey];
			if (currentState?.status === "running") {
				return;
			}

			// If tabId provided, associate this session with the tab
			if (tabId) {
				setTabSession(projectId, tabId, sessionKey);
			}

			bindBackendSession(baseSessionKey, sessionKey);

			setDocState(sessionKey, {
				projectId,
				projectName,
				sessionKey,
				events: [],
				todos: [],
				error: null,
				result: null,
				status: "running",
				cancellationRequested: false,
				activeTab: "activity",
				commitCompleted: false,
				completedCommitInfo: null,
				sourceBranch,
				targetBranch,
				chatOpen: false,
				messages: [],
				initialDiffSignatures: null,
				changedSinceInitial: [],
				docsInCodeRepo: false,
				docsBranch: null,
				mergeInProgress: false,
			});
			setActiveSessionKey(projectKey, sessionKey);
			updateSessionMeta(sessionKey, {
				projectId,
				projectName,
				sourceBranch,
				targetBranch,
				status: "running",
			});

			clearSubscriptions(sessionKey);

			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = toolEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation tool event", error, payload);
				}
			});

			const doneUnsub = EventsOn("events:llm:done", (payload) => {
				try {
					const evt = toolEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation done event", error, payload);
				}
			});

			const todoUnsub = EventsOn("event:llm:todo", (payload) => {
				try {
					const evt = todoEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						todos: evt.todos,
					}));
				} catch (error) {
					console.error("Invalid todo event", error, payload);
				}
			});

			subscriptions.set(sessionKey, {
				tool: toolUnsub,
				done: doneUnsub,
				todo: todoUnsub,
			});

			try {
				const result = await GenerateDocs(
					projectId,
					sourceBranch,
					targetBranch,
					modelKey,
					userInstructions,
					"",
					sessionKey
				);
				setDocState(sessionKey, {
					result,
					status: "success",
					cancellationRequested: false,
					initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
					changedSinceInitial: [],
					docsInCodeRepo: Boolean(result?.docsInCodeRepo),
					docsBranch: result?.docsBranch ?? null,
					mergeInProgress: false,
				});
				updateSessionMeta(sessionKey, { status: "success" });
			} catch (error) {
				const message = messageFromError(error);

				// Check for conflict with suggestion (new format)
				const suggestion = extractBranchConflictSuggestion(message);
				if (suggestion) {
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: null,
						status: "idle",
						conflict: {
							existingDocsBranch: suggestion.existing,
							proposedDocsBranch: suggestion.proposed,
							mode: "diff",
							isInProgress: suggestion.isInProgress,
						},
						activeTab: "activity",
					}));
					updateSessionMeta(sessionKey, { status: "idle" });
					return;
				}

				// Detect conflict (old format, backward compatibility)
				if (message.startsWith("ERR_DOCS_BRANCH_EXISTS:")) {
					const existing =
						extractExistingDocsBranch(message) ?? `docs/${sourceBranch}`;
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: null,
						status: "idle",
						conflict: {
							existingDocsBranch: existing,
							proposedDocsBranch: existing,
							mode: "diff",
						},
						activeTab: "activity",
					}));
					updateSessionMeta(sessionKey, { status: "idle" });
					return;
				}
				// ...existing non-conflict error handling...
				const normalized = message.toLowerCase();
				const docState = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
				const canceled =
					docState.cancellationRequested ||
					normalized.includes("context canceled") ||
					normalized.includes("context cancelled") ||
					normalized.includes("cancelled") ||
					normalized.includes("canceled");
				if (canceled) {
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: null,
						result: null,
						status: "canceled",
						cancellationRequested: false,
						events: [
							...prev.events,
							createLocalEvent(
								"warn",
								"Documentation generation canceled by user."
							),
						],
					}));
					updateSessionMeta(sessionKey, { status: "canceled" });
				} else {
					setDocState(sessionKey, {
						error: mapErrorCodeToMessage(message),
						status: "error",
						cancellationRequested: false,
						result: null,
						commitCompleted: false,
					});
					updateSessionMeta(sessionKey, { status: "error" });
				}
			} finally {
				unbindBackendSession(baseSessionKey, sessionKey);
				clearSubscriptions(sessionKey);
				setDocState(sessionKey, { cancellationRequested: false });
			}
		},

		startFromBranch: async ({
			projectId,
			projectName,
			sourceBranch,
			modelKey,
			userInstructions,
			tabId,
		}: StartArgs & { tabId?: TabId }) => {
			const projectKey = toKey(projectId);
			const baseSessionKey = createSessionKey(projectId, sourceBranch);
			const sessionKey = tabId
				? createSessionKey(projectId, sourceBranch, tabId)
				: baseSessionKey;
			const currentState = get().docStates[sessionKey];
			if (currentState?.status === "running") {
				return;
			}

			// If tabId provided, associate this session with the tab
			if (tabId) {
				setTabSession(projectId, tabId, sessionKey);
			}

			bindBackendSession(baseSessionKey, sessionKey);

			setDocState(sessionKey, {
				projectId,
				projectName,
				sessionKey,
				events: [],
				todos: [],
				error: null,
				result: null,
				status: "running",
				cancellationRequested: false,
				activeTab: "activity",
				commitCompleted: false,
				completedCommitInfo: null,
				sourceBranch,
				targetBranch: null,
				chatOpen: false,
				messages: [],
				initialDiffSignatures: null,
				changedSinceInitial: [],
				docsInCodeRepo: false,
				docsBranch: null,
				mergeInProgress: false,
			});
			setActiveSessionKey(projectKey, sessionKey);
			updateSessionMeta(sessionKey, {
				projectId,
				projectName,
				sourceBranch,
				targetBranch: "",
				status: "running",
			});

			clearSubscriptions(sessionKey);

			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = toolEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation tool event", error, payload);
				}
			});

			const doneUnsub = EventsOn("events:llm:done", (payload) => {
				try {
					const evt = toolEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation done event", error, payload);
				}
			});

			const todoUnsub = EventsOn("event:llm:todo", (payload) => {
				try {
					const evt = todoEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						todos: evt.todos,
					}));
				} catch (error) {
					console.error("Invalid todo event", error, payload);
				}
			});

			subscriptions.set(sessionKey, {
				tool: toolUnsub,
				done: doneUnsub,
				todo: todoUnsub,
			});

			try {
				const result = await GenerateDocsFromBranch(
					projectId,
					sourceBranch,
					modelKey,
					userInstructions,
					"",
					sessionKey
				);
				setDocState(sessionKey, {
					result,
					status: "success",
					cancellationRequested: false,
					initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
					changedSinceInitial: [],
					docsInCodeRepo: Boolean(result?.docsInCodeRepo),
					docsBranch: result?.docsBranch ?? null,
					mergeInProgress: false,
				});
				updateSessionMeta(sessionKey, { status: "success" });
			} catch (error) {
				const message = messageFromError(error);

				// Check for conflict with suggestion (new format)
				const suggestion = extractBranchConflictSuggestion(message);
				if (suggestion) {
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: null,
						status: "idle",
						conflict: {
							existingDocsBranch: suggestion.existing,
							proposedDocsBranch: suggestion.proposed,
							mode: "single",
							isInProgress: suggestion.isInProgress,
						},
						activeTab: "activity",
					}));
					updateSessionMeta(sessionKey, { status: "idle" });
					return;
				}

				// Detect conflict (old format, backward compatibility)
				if (message.startsWith("ERR_DOCS_BRANCH_EXISTS:")) {
					const existing =
						extractExistingDocsBranch(message) ?? `docs/${sourceBranch}`;
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: null,
						status: "idle",
						conflict: {
							existingDocsBranch: existing,
							proposedDocsBranch: existing,
							mode: "single",
						},
						activeTab: "activity",
					}));
					updateSessionMeta(sessionKey, { status: "idle" });
					return;
				}
				// ...existing non-conflict handling...
				const normalized = message.toLowerCase();
				const docState = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
				const canceled =
					docState.cancellationRequested ||
					normalized.includes("context canceled") ||
					normalized.includes("context cancelled") ||
					normalized.includes("cancelled") ||
					normalized.includes("canceled");
				if (canceled) {
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: null,
						result: null,
						status: "canceled",
						cancellationRequested: false,
						events: [
							...prev.events,
							createLocalEvent(
								"warn",
								"Documentation generation canceled by user."
							),
						],
					}));
					updateSessionMeta(sessionKey, { status: "canceled" });
				} else {
					setDocState(sessionKey, {
						error: mapErrorCodeToMessage(message),
						status: "error",
						cancellationRequested: false,
						result: null,
						commitCompleted: false,
					});
					updateSessionMeta(sessionKey, { status: "error" });
				}
			} finally {
				unbindBackendSession(baseSessionKey, sessionKey);
				clearSubscriptions(sessionKey);
				setDocState(sessionKey, { cancellationRequested: false });
			}
		},

		cancel: async (
			projectId: number | string,
			sessionKeyParam?: SessionKey
		) => {
			// Support both old (projectId) and new (sessionKey) approaches
			let sessionKey: SessionKey | null = null;
			let docState: DocGenerationData | null = null;

			if (sessionKeyParam) {
				sessionKey = sessionKeyParam;
				docState = get().docStates[sessionKey] ?? null;
			} else {
				const projectKey = toKey(projectId);
				const activeSessionKey = get().activeSession[projectKey];
				if (activeSessionKey) {
					sessionKey = activeSessionKey;
					docState = get().docStates[sessionKey] ?? null;
				}
			}

			if (!(sessionKey && docState)) {
				return;
			}

			const branch = docState.sourceBranch ?? "";
			const allowCancel =
				docState.status === "running" ||
				Boolean(docState.conflict?.isInProgress);
			if (!(allowCancel && branch)) {
				return;
			}

			setDocState(sessionKey, { cancellationRequested: true });
			try {
				await StopStream(Number(projectId), branch, sessionKey);
				updateSessionMeta(sessionKey, { status: "canceled" });
				setDocState(sessionKey, (prev) => ({
					...prev,
					cancellationRequested: false,
					status: "canceled",
					error: null,
					events: [
						...prev.events,
						createLocalEvent(
							"warn",
							"Documentation generation canceled by user."
						),
					],
				}));
			} catch (error) {
				const message = messageFromError(error);
				console.error("Failed to cancel doc generation", error);
				setDocState(sessionKey, (prev) => ({
					...prev,
					cancellationRequested: false,
					error: message,
					status: "error",
					result: null,
					events: [
						...prev.events,
						createLocalEvent(
							"error",
							`Failed to cancel documentation generation: ${message}`
						),
					],
				}));
				updateSessionMeta(sessionKey, { status: "error" });
			}
		},

		commit: async ({
			projectId,
			branch,
			files,
			sessionKey: sessionKeyParam,
		}: CommitArgs & { sessionKey?: SessionKey }) => {
			// Support both old (projectId) and new (sessionKey) approaches
			let sessionKey: SessionKey | null = null;

			if (sessionKeyParam) {
				sessionKey = sessionKeyParam;
			} else {
				const projectKey = toKey(projectId);
				sessionKey = get().activeSession[projectKey] ?? null;
			}

			if (!sessionKey) {
				return;
			}

			const docState = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
			if (docState.status === "committing") {
				return;
			}

			const label =
				docState.docsBranch && docState.docsBranch.trim() !== ""
					? docState.docsBranch
					: branch;

			setDocState(sessionKey, (prev) => ({
				...prev,
				error: null,
				status: "committing",
				events: [
					...prev.events,
					createLocalEvent(
						"info",
						`Committing documentation updates to ${label}`
					),
				],
				activeTab: "activity",
				commitCompleted: false,
			}));
			updateSessionMeta(sessionKey, (prev) => ({
				projectId: prev?.projectId ?? docState.projectId,
				projectName: prev?.projectName ?? docState.projectName,
				sourceBranch: prev?.sourceBranch ?? docState.sourceBranch ?? "",
				targetBranch: prev?.targetBranch ?? docState.targetBranch ?? "",
				status: "committing",
			}));

			try {
				await CommitDocs(projectId, branch, files);
				setDocState(sessionKey, (prev) => ({
					...prev,
					error: null,
					status: "success",
					events: [
						...prev.events,
						createLocalEvent(
							"info",
							`Committed documentation changes for ${label}`
						),
					],
					commitCompleted: true,
				}));
				updateSessionMeta(sessionKey, { status: "success" });
			} catch (error) {
				const message = messageFromError(error);
				setDocState(sessionKey, (prev) => ({
					...prev,
					error: message,
					status: "error",
					events: [
						...prev.events,
						createLocalEvent(
							"error",
							`Failed to commit documentation changes: ${message}`
						),
					],
					commitCompleted: false,
				}));
				updateSessionMeta(sessionKey, { status: "error" });
			}
		},

		setActiveSession: (projectId, sessionKey) => {
			setActiveSessionKey(toKey(projectId), sessionKey);
		},

		clearSessionMeta: (projectId, sourceBranch) => {
			const baseSessionKey = createSessionKey(projectId, sourceBranch);
			set((state) => {
				const nextMeta = { ...state.sessionMeta };
				for (const key of Object.keys(nextMeta)) {
					if (key === baseSessionKey || key.startsWith(`${baseSessionKey}:`)) {
						delete nextMeta[key];
					}
				}
				return { sessionMeta: nextMeta };
			});
		},

		mergeDocs: async ({
			projectId,
			branch,
			sessionKey: sessionKeyParam,
		}: {
			projectId: number;
			branch: string;
			sessionKey?: SessionKey;
		}): Promise<void> => {
			// Support both old (projectId) and new (sessionKey) approaches
			let sessionKey: SessionKey | null = null;

			if (sessionKeyParam) {
				sessionKey = sessionKeyParam;
			} else {
				const projectKey = toKey(projectId);
				sessionKey = get().activeSession[projectKey] ?? null;
			}

			if (!sessionKey) {
				return;
			}

			const docState = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
			if (!docState.docsInCodeRepo || docState.mergeInProgress) {
				return;
			}

			setDocState(sessionKey, (prev) => ({
				...prev,
				mergeInProgress: true,
				error: null,
				events: [
					...prev.events,
					createLocalEvent(
						"info",
						`Merging documentation branch into ${branch}`
					),
				],
			}));

			try {
				await MergeDocsIntoSource(projectId, branch);
				setDocState(sessionKey, (prev) => ({
					...prev,
					mergeInProgress: false,
					error: null,
					events: [
						...prev.events,
						createLocalEvent(
							"info",
							`Merged documentation updates into ${branch}`
						),
					],
					commitCompleted: true,
					completedCommitInfo: {
						sourceBranch: branch,
						targetBranch: prev.targetBranch ?? "",
						wasMerge: true,
					},
				}));
			} catch (error) {
				const rawMessage = messageFromError(error);
				const message = mapErrorCodeToMessage(rawMessage);
				setDocState(sessionKey, (prev) => ({
					...prev,
					mergeInProgress: false,
					error: message,
					events: [
						...prev.events,
						createLocalEvent(
							"error",
							`Failed to merge documentation branch: ${message}`
						),
					],
					commitCompleted: false,
				}));
			}
		},

		reset: (projectId: number | string, sessionKeyParam?: SessionKey) => {
			// Support both old (projectId) and new (sessionKey) approaches
			let sessionKey: SessionKey | null = null;

			if (sessionKeyParam) {
				// New approach: sessionKey provided
				sessionKey = sessionKeyParam;
			} else {
				// Old approach: find session from activeSession
				const projectKey = toKey(projectId);
				sessionKey = get().activeSession[projectKey] ?? null;
			}

			if (!sessionKey) {
				return;
			}

			clearSubscriptions(sessionKey);
			const current = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
			removeSessionMeta(sessionKey);

			setDocState(sessionKey, {
				...current,
				events: [],
				todos: [],
				error: null,
				result: null,
				status: "idle",
				cancellationRequested: false,
				activeTab: "activity",
				commitCompleted: false,
				completedCommitInfo: null,
				sourceBranch: null,
				targetBranch: null,
				chatOpen: false,
				messages: [],
				initialDiffSignatures: null,
				changedSinceInitial: [],
				docsInCodeRepo: false,
				docsBranch: null,
				mergeInProgress: false,
			});
		},

		setActiveTab: (sessionKey, tab) => {
			setDocState(sessionKey, { activeTab: tab });
		},

		setCommitCompleted: (sessionKey, completed) => {
			setDocState(sessionKey, { commitCompleted: completed });
		},

		setCompletedCommitInfo: (sessionKey, info) => {
			setDocState(sessionKey, { completedCommitInfo: info });
		},

		toggleChat: (sessionKey, open) => {
			setDocState(sessionKey, (prev) => ({
				...prev,
				chatOpen: typeof open === "boolean" ? open : !prev.chatOpen,
			}));
		},

		refine: async ({
			projectId,
			branch,
			instruction,
			sessionKey: sessionKeyParam,
		}) => {
			const trimmed = instruction.trim();
			if (!trimmed) {
				return;
			}

			// Support both old (projectId) and new (sessionKey) approaches
			let sessionKey: SessionKey | null = null;

			if (sessionKeyParam) {
				sessionKey = sessionKeyParam;
			} else {
				const projectKey = toKey(projectId);
				sessionKey = get().activeSession[projectKey] ?? null;
				// Fallback: try to find by branch
				if (!sessionKey) {
					sessionKey = createSessionKey(projectId, branch);
				}
			}

			if (!sessionKey) {
				return;
			}

			const docState = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
			if (docState.status === "running") {
				return;
			}
			const baseSessionKey = createSessionKey(projectId, branch);

			const messageId =
				typeof crypto !== "undefined" && "randomUUID" in crypto
					? crypto.randomUUID()
					: Math.random().toString(36).slice(2);
			const userMessage: ChatMessage = {
				id: messageId,
				role: "user",
				content: trimmed,
				status: "pending" as const,
				createdAt: new Date(),
			};

			setDocState(sessionKey, (prev) => ({
				...prev,
				messages: [...prev.messages, userMessage],
				error: null,
				status: "running",
			}));

			bindBackendSession(baseSessionKey, sessionKey);
			clearSubscriptions(sessionKey);
			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = toolEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid refine tool event", error, payload);
				}
			});
			const doneUnsub = EventsOn("events:llm:done", (payload) => {
				try {
					const evt = toolEventSchema.parse(payload);
					if (!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)) {
						return;
					}
					setDocState(sessionKey, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid refine done event", error, payload);
				}
			});

			subscriptions.set(sessionKey, { tool: toolUnsub, done: doneUnsub });

			try {
				const result = await RefineDocs(projectId, branch, trimmed, sessionKey);
				setDocState(sessionKey, (prev) => {
					const baseline =
						prev.initialDiffSignatures ??
						computeDiffSignatures(prev.result?.diff ?? null);
					const current = computeDiffSignatures(result?.diff ?? null);
					const changed = Object.keys(current).filter((path) => {
						const previousSignature = baseline[path];
						if (previousSignature === undefined) {
							return true;
						}
						return previousSignature !== current[path];
					});
					const updatedMessages = prev.messages.map<ChatMessage>((message) =>
						message.id === messageId
							? { ...message, status: "sent" as const }
							: message
					);
					const summary = (result?.summary ?? "").trim();
					const assistantMessages: ChatMessage[] = summary
						? [
								{
									id:
										typeof crypto !== "undefined" && "randomUUID" in crypto
											? crypto.randomUUID()
											: Math.random().toString(36).slice(2),
									role: "assistant",
									content: summary,
									createdAt: new Date(),
								},
							]
						: [];

					return {
						...prev,
						messages: [...updatedMessages, ...assistantMessages],
						result,
						status: "success",
						events: [
							...prev.events,
							createLocalEvent(
								"info",
								"Applied user instruction to documentation."
							),
						],
						initialDiffSignatures: baseline,
						changedSinceInitial: changed,
						cancellationRequested: false,
						docsInCodeRepo: Boolean(
							result?.docsInCodeRepo ?? prev.docsInCodeRepo
						),
						docsBranch: result?.docsBranch ?? prev.docsBranch,
						mergeInProgress: false,
					};
				});
			} catch (error) {
				const message = messageFromError(error);
				setDocState(sessionKey, (prev) => {
					const updatedMessages = prev.messages.map<ChatMessage>((msg) =>
						msg.id === messageId ? { ...msg, status: "error" as const } : msg
					);
					const assistantMessages: ChatMessage[] = [
						{
							id:
								typeof crypto !== "undefined" && "randomUUID" in crypto
									? crypto.randomUUID()
									: Math.random().toString(36).slice(2),
							role: "assistant",
							content: `Error: ${message}`,
							createdAt: new Date(),
						},
					];
					return {
						...prev,
						messages: [...updatedMessages, ...assistantMessages],
						error: message,
						status: "error",
						events: [
							...prev.events,
							createLocalEvent(
								"error",
								`Failed to refine documentation: ${message}`
							),
						],
					};
				});
			} finally {
				unbindBackendSession(baseSessionKey, sessionKey);
				clearSubscriptions(sessionKey);
				setDocState(sessionKey, { cancellationRequested: false });
			}
		},

		restoreSession: async (
			projectId: number,
			sourceBranch: string,
			targetBranch: string,
			tabId?: TabId
		): Promise<boolean> => {
			const projectKey = toKey(projectId);
			const baseSessionKey = createSessionKey(projectId, sourceBranch);
			const sessionKey = tabId
				? createSessionKey(projectId, sourceBranch, tabId)
				: baseSessionKey;
			const currentState = get().docStates[sessionKey];

			// If tabId provided, associate this session with the tab
			if (tabId) {
				setTabSession(projectId, tabId, sessionKey);
			}

			// Only reject if there's an existing running/success state for this specific session
			if (
				currentState &&
				(currentState.status === "running" ||
					(currentState.status === "success" && currentState.result))
			) {
				return false;
			}

			try {
				const sessionMeta = get().sessionMeta[sessionKey];
				const projectName =
					currentState?.projectName ?? sessionMeta?.projectName ?? "";

				// Get project name from repo if needed
				let finalProjectName = projectName;
				if (!finalProjectName) {
					try {
						const repoLink = await GetRepoLink(projectId);
						finalProjectName = repoLink?.ProjectName ?? "";
					} catch {
						// Ignore error, projectName can be empty
					}
				}

				setDocState(sessionKey, {
					projectId,
					projectName: finalProjectName,
					sessionKey,
					status: "running",
					events: [
						createLocalEvent(
							"info",
							`Restoring documentation session for branches: ${sourceBranch} → ${targetBranch}`
						),
					],
					error: null,
					result: null,
					cancellationRequested: false,
					activeTab: "activity",
					commitCompleted: false,
					completedCommitInfo: null,
					sourceBranch: sourceBranch || null,
					targetBranch: targetBranch || null,
					chatOpen: false,
					messages: [],
					initialDiffSignatures: null,
					changedSinceInitial: [],
					docsInCodeRepo: false,
					docsBranch: null,
					mergeInProgress: false,
				});

				const result = await LoadGenerationSession(
					projectId,
					sourceBranch,
					targetBranch
				);

				setDocState(sessionKey, {
					projectId,
					projectName: finalProjectName,
					sessionKey,
					result,
					status: "success",
					sourceBranch: sourceBranch || null,
					targetBranch: targetBranch || null,
					initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
					changedSinceInitial: [],
					docsInCodeRepo: result?.docsInCodeRepo,
					docsBranch: result?.docsBranch ?? null,
					events: [
						createLocalEvent(
							"info",
							`Session restored successfully - ${result.files?.length ?? 0} file(s) modified`
						),
					],
					activeTab: "review",
					error: null,
					cancellationRequested: false,
					commitCompleted: false,
					completedCommitInfo: null,
					chatOpen: false,
					messages: [],
					mergeInProgress: false,
				});
				setActiveSessionKey(projectKey, sessionKey);
				updateSessionMeta(sessionKey, {
					projectId,
					projectName: finalProjectName,
					sourceBranch,
					targetBranch,
					status: "success",
				});
				return true;
			} catch (error) {
				const message = messageFromError(error);
				console.error("Failed to restore generation session", error);
				setDocState(sessionKey, {
					projectId,
					projectName: currentState?.projectName ?? "",
					sessionKey,
					status: "idle",
					events: [
						createLocalEvent("warn", `Could not restore session: ${message}`),
					],
					error: message,
					result: null,
					cancellationRequested: false,
					activeTab: "activity",
					commitCompleted: false,
					completedCommitInfo: null,
					sourceBranch: null,
					targetBranch: null,
					chatOpen: false,
					messages: [],
					initialDiffSignatures: null,
					changedSinceInitial: [],
					docsInCodeRepo: false,
					docsBranch: null,
					mergeInProgress: false,
				});
				return false;
			}
		},

		clearConflict: (sessionKey: SessionKey) => {
			setDocState(sessionKey, { conflict: null, error: null });
		},

		resolveDocsBranchConflictByDelete: async ({
			projectId,
			projectName,
			sourceBranch,
			mode,
			targetBranch,
			modelKey,
			userInstructions,
			tabId,
			sessionKey: sessionKeyOverride,
		}) => {
			const sessionKey =
				sessionKeyOverride ??
				(tabId
					? createSessionKey(projectId, sourceBranch, tabId)
					: createSessionKey(projectId, sourceBranch));
			const state = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
			const existing = state.conflict?.existingDocsBranch?.trim();
			if (!existing) {
				// Nothing to resolve; just clear and return
				setDocState(sessionKey, { conflict: null });
				return;
			}
			try {
				setDocState(sessionKey, (prev) => ({
					...prev,
					error: null,
					events: [
						...prev.events,
						createLocalEvent(
							"info",
							`Deleting existing docs branch '${existing}'`
						),
					],
				}));
				const project = await GetRepoLink(projectId);
				const repoRoot = (project?.DocumentationRepo ?? "").trim();
				if (!repoRoot) {
					return Promise.reject(
						new Error("Documentation repository path is not configured")
					);
				}
				await DeleteBranchByPath(repoRoot, existing);

				// Clear the conflict before restarting
				setDocState(sessionKey, { conflict: null });

				if (mode === "diff") {
					await get().start({
						projectId,
						projectName,
						sourceBranch,
						targetBranch: targetBranch ?? "",
						modelKey,
						userInstructions,
						tabId,
					});
				} else {
					await get().startFromBranch?.({
						projectId,
						projectName,
						sourceBranch,
						targetBranch: "",
						modelKey,
						userInstructions,
						tabId,
					} as StartArgs & { tabId?: TabId });
				}
			} catch (error) {
				const message = messageFromError(error);
				setDocState(sessionKey, (prev) => ({
					...prev,
					error: mapErrorCodeToMessage(message),
					status: "idle",
					// Keep the conflict so the dialog remains open
					conflict: prev.conflict,
					// Avoid adding events so the session UI doesn't appear in background
					events: [],
				}));
			}
		},

		resolveDocsBranchConflictByRename: async ({
			projectId,
			sourceBranch,
			newDocsBranch,
			mode,
			targetBranch,
			modelKey,
			userInstructions,
			tabId,
			sessionKey: sessionKeyOverride,
		}) => {
			const baseSessionKey = createSessionKey(projectId, sourceBranch);
			const sessionKey =
				sessionKeyOverride ??
				(tabId
					? createSessionKey(projectId, sourceBranch, tabId)
					: baseSessionKey);
			const state = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
			const existing = state.conflict?.existingDocsBranch?.trim();
			const targetName = (newDocsBranch ?? "").trim();
			if (!(existing && targetName)) {
				return;
			}
			try {
				// If tabId provided, associate this session with the tab
				if (tabId) {
					setTabSession(projectId, tabId, sessionKey);
				}

				// Transition to running without wiping the prior activity feed
				setDocState(sessionKey, (prev) => ({
					...prev,
					projectId,
					projectName: state.projectName,
					sessionKey,
					activeTab: "activity",
					status: "running",
					sourceBranch,
					targetBranch:
						mode === "diff"
							? (targetBranch ?? prev.targetBranch ?? null)
							: (prev.targetBranch ?? null),
					result: null,
					error: null,
					cancellationRequested: false,
					docsInCodeRepo: false,
					docsBranch: null,
					mergeInProgress: false,
					initialDiffSignatures: null,
					changedSinceInitial: [],
					todos: [],
					events: [
						...prev.events,
						createLocalEvent(
							"info",
							`Using new docs branch name '${targetName}'`
						),
					],
				}));

				bindBackendSession(baseSessionKey, sessionKey);
				clearSubscriptions(sessionKey);
				const toolUnsub = EventsOn("event:llm:tool", (payload) => {
					try {
						const evt = toolEventSchema.parse(payload);
						if (
							!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)
						) {
							return;
						}
						setDocState(sessionKey, (prev) => ({
							...prev,
							events: [...prev.events, evt],
						}));
					} catch (error) {
						console.error("Invalid refine tool event", error, payload);
					}
				});
				const doneUnsub = EventsOn("events:llm:done", (payload) => {
					try {
						const evt = toolEventSchema.parse(payload);
						if (
							!isEventForSession(evt.sessionKey, sessionKey, baseSessionKey)
						) {
							return;
						}
						setDocState(sessionKey, (prev) => ({
							...prev,
							events: [...prev.events, evt],
						}));
					} catch {}
				});
				subscriptions.set(sessionKey, { tool: toolUnsub, done: doneUnsub });

				// Call backend to generate directly into the provided docs branch name
				let result: models.DocGenerationResult | null = null;
				if (mode === "diff") {
					if (!targetBranch) {
						return Promise.reject(
							new Error("Target branch is required for diff mode")
						);
					}
					result = await GenerateDocs(
						projectId,
						sourceBranch,
						targetBranch,
						modelKey,
						userInstructions,
						targetName,
						sessionKey
					);
				} else {
					result = await GenerateDocsFromBranch(
						projectId,
						sourceBranch,
						modelKey,
						userInstructions,
						targetName,
						sessionKey
					);
				}

				// Success: populate result and status, clear conflict
				setDocState(sessionKey, (prev) => ({
					...prev,
					result: result ?? prev.result,
					status: "success",
					cancellationRequested: false,
					initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
					changedSinceInitial: [],
					docsInCodeRepo: Boolean(
						result?.docsInCodeRepo ?? prev.docsInCodeRepo
					),
					docsBranch: result?.docsBranch ?? targetName,
					mergeInProgress: false,
					conflict: null,
				}));
				updateSessionMeta(sessionKey, { status: "success" });
			} catch (error) {
				// On failure, ensure any in-flight generation is stopped
				try {
					await StopStream(Number(projectId), sourceBranch ?? "", sessionKey);
				} catch {}
				const message = messageFromError(error);
				const normalized = message.toLowerCase();
				const current = get().docStates[sessionKey] ?? EMPTY_DOC_STATE;
				const canceled =
					current.cancellationRequested ||
					normalized.includes("context canceled") ||
					normalized.includes("context cancelled") ||
					normalized.includes("cancelled") ||
					normalized.includes("canceled");
				if (canceled) {
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: null,
						status: "canceled",
						cancellationRequested: false,
						events: [
							...prev.events,
							createLocalEvent(
								"warn",
								"Documentation generation canceled by user."
							),
						],
					}));
					updateSessionMeta(sessionKey, { status: "canceled" });
				} else {
					setDocState(sessionKey, (prev) => ({
						...prev,
						error: mapErrorCodeToMessage(message),
						status: "error",
						// Keep conflict so the dialog stays present and allow user to try another name
						conflict: prev.conflict
							? { ...prev.conflict, proposedDocsBranch: targetName }
							: prev.conflict,
						events: [
							...prev.events,
							createLocalEvent(
								"error",
								`Failed to create docs branch '${targetName}': ${message}`
							),
						],
					}));
					updateSessionMeta(sessionKey, { status: "error" });
				}
			} finally {
				unbindBackendSession(baseSessionKey, sessionKey);
				clearSubscriptions(sessionKey);
				setDocState(sessionKey, { cancellationRequested: false });
			}
		},
	};
});
