import type { models, services } from "@go/models";
import {
	CheckDocsBranchAvailability,
	CommitDocs,
	GenerateDocs,
	GenerateDocsFromBranch,
	GetAvailableTabSessions,
	LoadGenerationSession,
	MergeDocsIntoSource,
	RefineDocs,
	StopStream,
} from "@go/services/ClientService";
import { DeleteBranchByPath } from "@go/services/GitService";
import {
	DeleteByID as DeleteGenerationSession,
	GetByDocsBranch,
} from "@go/services/generationSessionService";
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
	files: string[];
};

type ProjectKey = string;
export type SessionKey = string;
type TabId = string;

type SessionMeta = {
	sessionId: number | null;
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

export type ChatMessage = {
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
	sessionId: number | null;
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
	reset: (
		sessionKey: SessionKey,
		options?: { deleteDocsBranch?: boolean }
	) => Promise<void>;
	commit: (args: CommitArgs & { sessionKey: SessionKey }) => Promise<void>;
	cancel: (sessionKey: SessionKey) => Promise<void>;
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
		sessionKey: SessionKey;
		instruction: string;
	}) => Promise<void>;
	mergeDocs: (args: { sessionKey: SessionKey }) => Promise<void>;
	restoreSession: (
		sessionInfo: services.SessionInfo,
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
	sessionId: null,
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

// Create a session key from a session ID (used after backend creates session)
export const createSessionKey = (
	sessionId: number,
	tabId?: TabId | null
): SessionKey => {
	const baseKey = `session:${sessionId}`;
	const normalizedTab = normalizeTabId(tabId);
	return normalizedTab ? `${baseKey}:${normalizedTab}` : baseKey;
};

// Create a temporary session key before we have a session ID (for pre-generation UI state)
export const createTempSessionKey = (
	projectId: number,
	sourceBranch: string | null | undefined,
	tabId?: TabId | null
): SessionKey => {
	const normalizedBranch = (sourceBranch ?? "").trim();
	const baseKey = `temp:${projectId}:${normalizedBranch}`;
	const normalizedTab = normalizeTabId(tabId);
	return normalizedTab ? `${baseKey}:${normalizedTab}` : baseKey;
};

const WHITESPACE_REGEX = /\s+/g;

const documentationBranchName = (
	sourceBranch: string | null | undefined
): string => {
	const trimmed = (sourceBranch ?? "").trim();
	if (!trimmed) {
		return "docs";
	}
	return `docs/${trimmed.replace(WHITESPACE_REGEX, "-")}`;
};

const backendSessionBindings = new Map<SessionKey, Set<SessionKey>>();

const bindBackendSession = (
	baseSessionKey: SessionKey,
	sessionKey: SessionKey
) => {
	const existing = backendSessionBindings.get(baseSessionKey) ?? new Set();
	existing.add(sessionKey);
	backendSessionBindings.set(baseSessionKey, existing);
};

const unbindBackendSession = (
	baseSessionKey: SessionKey,
	sessionKey?: SessionKey
) => {
	if (!sessionKey) {
		backendSessionBindings.delete(baseSessionKey);
		return;
	}
	const existing = backendSessionBindings.get(baseSessionKey);
	if (existing) {
		existing.delete(sessionKey);
		if (existing.size === 0) {
			backendSessionBindings.delete(baseSessionKey);
		}
	}
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
	const boundSessions = backendSessionBindings.get(baseSessionKey);
	if (boundSessions) {
		return boundSessions.has(sessionKey);
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
		trimmed.startsWith("ERR_DOCS_GENERATION_IN_PROGRESS_SUGGEST:") ||
		trimmed.startsWith("ERR_SESSION_EXISTS_SUGGEST:")
	) {
		return i18n.t("common.docsBranchConflict");
	}

	if (
		trimmed.startsWith("ERR_DOCS_BRANCH_EXISTS:") ||
		trimmed.startsWith("ERR_DOCS_GENERATION_IN_PROGRESS:")
	) {
		return i18n.t("common.docsBranchExists");
	}

	if (trimmed.startsWith("ERR_SESSION_EXISTS:")) {
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
// - ERR_SESSION_EXISTS_SUGGEST:docs/feature:docs/feature-2
const extractBranchConflictSuggestion = (
	errorMessage: string
): { existing: string; proposed: string; isInProgress: boolean } | null => {
	const inProgressPrefix = "ERR_DOCS_GENERATION_IN_PROGRESS_SUGGEST:";
	const existsPrefix = "ERR_DOCS_BRANCH_EXISTS_SUGGEST:";
	const sessionExistsPrefix = "ERR_SESSION_EXISTS_SUGGEST:";

	let isInProgress = false;
	let remainder = "";

	if (errorMessage.startsWith(inProgressPrefix)) {
		isInProgress = true;
		remainder = errorMessage.slice(inProgressPrefix.length);
	} else if (errorMessage.startsWith(existsPrefix)) {
		isInProgress = false;
		remainder = errorMessage.slice(existsPrefix.length);
	} else if (errorMessage.startsWith(sessionExistsPrefix)) {
		isInProgress = false;
		remainder = errorMessage.slice(sessionExistsPrefix.length);
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

const normalizeChatMessages = (raw: unknown): ChatMessage[] => {
	if (!Array.isArray(raw)) {
		return [];
	}
	const normalized: ChatMessage[] = [];
	for (let i = 0; i < raw.length; i += 1) {
		const entry = raw[i] as Record<string, unknown>;
		const roleValue = typeof entry?.role === "string" ? entry.role.trim() : "";
		const role =
			roleValue === "assistant" || roleValue === "user" ? roleValue : null;
		const content =
			typeof entry?.content === "string" ? entry.content.trim() : "";
		if (!(role && content)) {
			continue;
		}
		const parsedDate =
			entry?.createdAt && typeof entry.createdAt === "string"
				? new Date(entry.createdAt)
				: new Date();
		const createdAt = Number.isNaN(parsedDate.getTime())
			? new Date()
			: parsedDate;
		const fallbackId = `chat-${role}-${createdAt.getTime()}-${i}`;
		const id =
			typeof crypto !== "undefined" && "randomUUID" in crypto
				? crypto.randomUUID()
				: fallbackId;
		normalized.push({
			id,
			role,
			content,
			createdAt,
			status: "sent",
		});
	}
	return normalized;
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

	const findTabIdForSession = (
		projectId: number,
		sessionKey: SessionKey
	): TabId | null => {
		const projectKey = toKey(projectId);
		const projectTabs = get().tabSessions[projectKey];
		if (!projectTabs) {
			return null;
		}
		for (const [tabId, mappedSession] of Object.entries(projectTabs)) {
			if (mappedSession === sessionKey) {
				return tabId;
			}
		}
		return null;
	};

	const subscribeToGenerationEvents = (
		sessionKey: SessionKey,
		baseSessionKey: SessionKey
	) => {
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
	};

	type GenerationMode = "diff" | "single";
	type RunGenerationArgs = {
		projectId: number;
		projectName: string;
		sourceBranch: string;
		targetBranch?: string;
		modelKey: string;
		userInstructions: string;
		tabId?: TabId;
		conflictMode: GenerationMode;
		runRequest: (sessionKey: SessionKey) => Promise<models.DocGenerationResult>;
	};

	const runGeneration = async ({
		projectId,
		projectName,
		sourceBranch,
		targetBranch,
		modelKey,
		userInstructions,
		tabId,
		conflictMode,
		runRequest,
	}: RunGenerationArgs) => {
		const projectKey = toKey(projectId);
		const docsBranch = documentationBranchName(sourceBranch);
		// Use temp session key until we get the real session ID from the backend
		const tempSessionKey = createTempSessionKey(projectId, sourceBranch, tabId);
		const currentState = get().docStates[tempSessionKey];
		if (currentState?.status === "running") {
			return;
		}

		if (tabId) {
			setTabSession(projectId, tabId, tempSessionKey);
		}

		// Pre-check for docs branch conflicts BEFORE setting UI to "running"
		try {
			await CheckDocsBranchAvailability(projectId, sourceBranch, "");
		} catch (checkError) {
			const checkMessage = messageFromError(checkError);
			const suggestion = extractBranchConflictSuggestion(checkMessage);
			if (suggestion) {
				// Set up session state for the conflict dialog (but NOT "running")
				setDocState(tempSessionKey, {
					sessionId: null,
					projectId,
					projectName,
					sessionKey: tempSessionKey,
					events: [],
					todos: [],
					error: null,
					result: null,
					status: "idle",
					cancellationRequested: false,
					activeTab: "activity",
					commitCompleted: false,
					completedCommitInfo: null,
					sourceBranch,
					targetBranch: targetBranch ?? null,
					chatOpen: false,
					messages: [],
					initialDiffSignatures: null,
					changedSinceInitial: [],
					docsInCodeRepo: false,
					docsBranch: suggestion.existing || docsBranch,
					mergeInProgress: false,
					conflict: {
						existingDocsBranch: suggestion.existing,
						proposedDocsBranch: suggestion.proposed,
						mode: conflictMode,
						isInProgress: suggestion.isInProgress,
					},
				});
				setActiveSessionKey(projectKey, tempSessionKey);
				updateSessionMeta(tempSessionKey, {
					sessionId: null,
					projectId,
					projectName,
					sourceBranch,
					targetBranch: targetBranch ?? "",
					status: "idle",
				});
				return;
			}

			if (checkMessage.startsWith("ERR_DOCS_BRANCH_EXISTS:")) {
				const existing = extractExistingDocsBranch(checkMessage) ?? docsBranch;
				setDocState(tempSessionKey, {
					sessionId: null,
					projectId,
					projectName,
					sessionKey: tempSessionKey,
					events: [],
					todos: [],
					error: null,
					result: null,
					status: "idle",
					cancellationRequested: false,
					activeTab: "activity",
					commitCompleted: false,
					completedCommitInfo: null,
					sourceBranch,
					targetBranch: targetBranch ?? null,
					chatOpen: false,
					messages: [],
					initialDiffSignatures: null,
					changedSinceInitial: [],
					docsInCodeRepo: false,
					docsBranch: existing,
					mergeInProgress: false,
					conflict: {
						existingDocsBranch: existing,
						proposedDocsBranch: existing,
						mode: conflictMode,
					},
				});
				setActiveSessionKey(projectKey, tempSessionKey);
				updateSessionMeta(tempSessionKey, {
					sessionId: null,
					projectId,
					projectName,
					sourceBranch,
					targetBranch: targetBranch ?? "",
					status: "idle",
				});
				return;
			}

			if (checkMessage.startsWith("ERR_SESSION_EXISTS:")) {
				// Handle session conflict like a branch conflict - show dialog
				// so user can delete the session/branch or use an alternative name
				setDocState(tempSessionKey, {
					sessionId: null,
					projectId,
					projectName,
					sessionKey: tempSessionKey,
					events: [],
					todos: [],
					error: null,
					result: null,
					status: "idle",
					cancellationRequested: false,
					activeTab: "activity",
					commitCompleted: false,
					completedCommitInfo: null,
					sourceBranch,
					targetBranch: targetBranch ?? null,
					chatOpen: false,
					messages: [],
					initialDiffSignatures: null,
					changedSinceInitial: [],
					docsInCodeRepo: false,
					docsBranch,
					mergeInProgress: false,
					conflict: {
						existingDocsBranch: docsBranch,
						proposedDocsBranch: docsBranch,
						mode: conflictMode,
					},
				});
				setActiveSessionKey(projectKey, tempSessionKey);
				updateSessionMeta(tempSessionKey, {
					sessionId: null,
					projectId,
					projectName,
					sourceBranch,
					targetBranch: targetBranch ?? "",
					status: "idle",
				});
				return;
			}
			// For other errors, let them fall through to the main generation flow
		}

		// Bind the temp key for event routing
		bindBackendSession(tempSessionKey, tempSessionKey);

		setDocState(tempSessionKey, {
			sessionId: null,
			projectId,
			projectName,
			sessionKey: tempSessionKey,
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
			targetBranch: targetBranch ?? null,
			chatOpen: false,
			messages: [],
			initialDiffSignatures: null,
			changedSinceInitial: [],
			docsInCodeRepo: false,
			docsBranch,
			mergeInProgress: false,
		});
		setActiveSessionKey(projectKey, tempSessionKey);
		updateSessionMeta(tempSessionKey, {
			sessionId: null,
			projectId,
			projectName,
			sourceBranch,
			targetBranch: targetBranch ?? "",
			status: "running",
		});

		subscribeToGenerationEvents(tempSessionKey, tempSessionKey);

		try {
			const result = await runRequest(tempSessionKey);
			// Extract sessionId from result and update state
			const sessionId = result?.sessionId ?? null;
			setDocState(tempSessionKey, {
				sessionId,
				result,
				status: "success",
				cancellationRequested: false,
				initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
				changedSinceInitial: [],
				docsInCodeRepo: Boolean(result?.docsInCodeRepo),
				docsBranch: result?.docsBranch ?? null,
				mergeInProgress: false,
				messages: normalizeChatMessages(result?.chatMessages ?? []),
			});
			updateSessionMeta(tempSessionKey, { sessionId, status: "success" });
		} catch (error) {
			const message = messageFromError(error);

			const suggestion = extractBranchConflictSuggestion(message);
			if (suggestion) {
				setDocState(tempSessionKey, (prev) => ({
					...prev,
					error: null,
					status: "idle",
					conflict: {
						existingDocsBranch: suggestion.existing,
						proposedDocsBranch: suggestion.proposed,
						mode: conflictMode,
						isInProgress: suggestion.isInProgress,
					},
					activeTab: "activity",
				}));
				updateSessionMeta(tempSessionKey, { status: "idle" });
				return;
			}

			if (message.startsWith("ERR_DOCS_BRANCH_EXISTS:")) {
				const existing =
					extractExistingDocsBranch(message) ?? `docs/${sourceBranch}`;
				setDocState(tempSessionKey, (prev) => ({
					...prev,
					error: null,
					status: "idle",
					conflict: {
						existingDocsBranch: existing,
						proposedDocsBranch: existing,
						mode: conflictMode,
					},
					activeTab: "activity",
				}));
				updateSessionMeta(tempSessionKey, { status: "idle" });
				return;
			}

			const normalized = message.toLowerCase();
			const docState = get().docStates[tempSessionKey] ?? EMPTY_DOC_STATE;
			const canceled =
				docState.cancellationRequested ||
				normalized.includes("context canceled") ||
				normalized.includes("context cancelled") ||
				normalized.includes("cancelled") ||
				normalized.includes("canceled");
			if (canceled) {
				setDocState(tempSessionKey, (prev) => ({
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
				updateSessionMeta(tempSessionKey, { status: "canceled" });
			} else {
				setDocState(tempSessionKey, {
					error: mapErrorCodeToMessage(message),
					status: "error",
					cancellationRequested: false,
					result: null,
					commitCompleted: false,
				});
				updateSessionMeta(tempSessionKey, { status: "error" });
			}
		} finally {
			unbindBackendSession(tempSessionKey, tempSessionKey);
			clearSubscriptions(tempSessionKey);
			setDocState(tempSessionKey, { cancellationRequested: false });
		}
	};

	const deriveDocsBranch = (state: DocGenerationData): string => {
		const conflictBranch = state.conflict?.existingDocsBranch?.trim();
		if (conflictBranch) {
			return conflictBranch;
		}
		const existing = state.docsBranch?.trim();
		if (existing) {
			return existing;
		}
		return documentationBranchName(state.sourceBranch);
	};

	const resolveSessionIdForKey = async (
		sessionKey: SessionKey,
		state: DocGenerationData
	): Promise<number | null> => {
		if (state.sessionId) {
			return state.sessionId;
		}
		const expectedDocsBranch = deriveDocsBranch(state);
		try {
			const sessions = await GetAvailableTabSessions(state.projectId);
			const match = sessions.find((session) => {
				const docsBranch = (session.docsBranch ?? "").trim();
				const source = (session.sourceBranch ?? "").trim();
				return (
					docsBranch === expectedDocsBranch &&
					source === (state.sourceBranch ?? "").trim()
				);
			});
			if (match?.id) {
				setDocState(sessionKey, {
					sessionId: match.id,
					docsBranch: expectedDocsBranch,
				});
				updateSessionMeta(sessionKey, { sessionId: match.id });
				return match.id;
			}
		} catch {
			return null;
		}
		return null;
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
			await runGeneration({
				projectId,
				projectName,
				sourceBranch,
				targetBranch,
				modelKey,
				userInstructions,
				tabId,
				conflictMode: "diff",
				runRequest: (sessionKey) =>
					GenerateDocs(
						projectId,
						sourceBranch,
						targetBranch,
						modelKey,
						userInstructions,
						"",
						sessionKey
					),
			});
		},

		startFromBranch: async ({
			projectId,
			projectName,
			sourceBranch,
			modelKey,
			userInstructions,
			tabId,
		}: StartArgs & { tabId?: TabId }) => {
			await runGeneration({
				projectId,
				projectName,
				sourceBranch,
				targetBranch: undefined,
				modelKey,
				userInstructions,
				tabId,
				conflictMode: "single",
				runRequest: (sessionKey) =>
					GenerateDocsFromBranch(
						projectId,
						sourceBranch,
						modelKey,
						userInstructions,
						"",
						sessionKey
					),
			});
		},

		cancel: async (sessionKey: SessionKey) => {
			const docState = get().docStates[sessionKey];
			if (!docState) {
				return;
			}

			const allowCancel =
				docState.status === "running" ||
				Boolean(docState.conflict?.isInProgress);
			if (!allowCancel) {
				return;
			}
			const sessionId =
				docState.sessionId ??
				(await resolveSessionIdForKey(sessionKey, docState));
			if (!sessionId) {
				setDocState(sessionKey, { cancellationRequested: false });
				return;
			}

			setDocState(sessionKey, { cancellationRequested: true });
			try {
				// StopStream now takes (sessionID, sessionKeyOverride)
				// Resolve the session ID before issuing the stop request
				await StopStream(sessionId, sessionKey);
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
			files,
			sessionKey,
		}: CommitArgs & { sessionKey: SessionKey }) => {
			const docState = get().docStates[sessionKey];
			if (!docState || docState.status === "committing") {
				return;
			}

			const sessionId = docState.sessionId;
			if (!sessionId) {
				console.error("Cannot commit: no session ID");
				return;
			}

			const branch = docState.sourceBranch ?? "";
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
				sessionId: prev?.sessionId ?? sessionId,
				projectId: prev?.projectId ?? docState.projectId,
				projectName: prev?.projectName ?? docState.projectName,
				sourceBranch: prev?.sourceBranch ?? docState.sourceBranch ?? "",
				targetBranch: prev?.targetBranch ?? docState.targetBranch ?? "",
				status: "committing",
			}));

			try {
				// CommitDocs now takes (projectID, sessionID, files)
				await CommitDocs(docState.projectId, sessionId, files);
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
			// Use temp session key pattern for clearing pre-session metadata
			const baseTempKey = createTempSessionKey(projectId, sourceBranch);
			set((state) => {
				const nextMeta = { ...state.sessionMeta };
				for (const key of Object.keys(nextMeta)) {
					// Clear both temp keys and any keys starting with the temp pattern
					if (key === baseTempKey || key.startsWith(`${baseTempKey}:`)) {
						delete nextMeta[key];
					}
				}
				return { sessionMeta: nextMeta };
			});
		},

		mergeDocs: async ({
			sessionKey,
		}: {
			sessionKey: SessionKey;
		}): Promise<void> => {
			const docState = get().docStates[sessionKey];
			if (!(docState && docState.docsInCodeRepo) || docState.mergeInProgress) {
				return;
			}

			const sessionId = docState.sessionId;
			if (!sessionId) {
				console.error("Cannot merge: no session ID");
				return;
			}

			const branch = docState.sourceBranch ?? "";

			setDocState(sessionKey, (prev) => ({
				...prev,
				mergeInProgress: true,
				error: null,
				events: [
					...prev.events,
					createLocalEvent(
						"info",
						`Merging documentation branch into ${branch || "source"}`
					),
				],
			}));

			try {
				// MergeDocsIntoSource now takes only sessionID
				await MergeDocsIntoSource(sessionId);
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

		reset: async (
			sessionKey: SessionKey,
			options?: { deleteDocsBranch?: boolean }
		) => {
			const current = get().docStates[sessionKey];
			if (!current) {
				return;
			}

			try {
				// Delete the docs branch if it exists (always true when called from UI)
				if (options?.deleteDocsBranch && current.docsBranch) {
					const project = await GetRepoLink(current.projectId);
					const repoRoot = current.docsInCodeRepo
						? (project?.CodebaseRepo ?? "")
						: (project?.DocumentationRepo ?? "");

					if (repoRoot && current.docsBranch) {
						try {
							await DeleteBranchByPath(repoRoot, current.docsBranch);
							setDocState(sessionKey, (prev) => ({
								...prev,
								events: [
									...prev.events,
									createLocalEvent(
										"info",
										`Deleted docs branch '${current.docsBranch}'`
									),
								],
							}));
						} catch (error) {
							console.error("Failed to delete docs branch", error);
							setDocState(sessionKey, (prev) => ({
								...prev,
								events: [
									...prev.events,
									createLocalEvent(
										"warn",
										`Failed to delete docs branch: ${messageFromError(error)}`
									),
								],
							}));
						}
					}
				}

				// Clean up backend session: deletes temp directories, wipes session state, and removes metadata
				const sessionId = current.sessionId;
				if (sessionId) {
					await DeleteGenerationSession(sessionId);
				}
			} catch (error) {
				console.error("Failed to clean up backend session", error);
			}

			clearSubscriptions(sessionKey);
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

		refine: async ({ sessionKey, instruction }) => {
			const trimmed = instruction.trim();
			if (!trimmed) {
				return;
			}

			const docState = get().docStates[sessionKey];
			if (!docState || docState.status === "running") {
				return;
			}

			const sessionId = docState.sessionId;
			if (!sessionId) {
				console.error("Cannot refine: no session ID");
				return;
			}

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

			bindBackendSession(sessionKey, sessionKey);
			subscribeToGenerationEvents(sessionKey, sessionKey);

			try {
				// RefineDocs now takes (sessionID, instruction, sessionKeyOverride)
				const result = await RefineDocs(sessionId, trimmed, sessionKey);
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
									status: "sent",
								},
							]
						: [];
					const persistedChat = normalizeChatMessages(
						(result as any)?.chatMessages
					);
					const chatMessages =
						persistedChat.length > 0
							? persistedChat
							: [...updatedMessages, ...assistantMessages];
					const nextResult = result
						? ({
								...result,
								chatMessages: (result as any)?.chatMessages,
							} as models.DocGenerationResult)
						: result;

					return {
						...prev,
						messages: chatMessages,
						result: nextResult,
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
				unbindBackendSession(sessionKey, sessionKey);
				clearSubscriptions(sessionKey);
				setDocState(sessionKey, { cancellationRequested: false });
			}
		},

		restoreSession: async (
			sessionInfo: services.SessionInfo,
			tabId?: TabId
		): Promise<boolean> => {
			const {
				id: sessionId,
				projectId,
				sourceBranch,
				targetBranch,
			} = sessionInfo;
			if (!sessionId) {
				console.error("Cannot restore session: no session ID");
				return false;
			}

			const projectKey = toKey(projectId);
			// Use the real session key based on ID
			const sessionKey = createSessionKey(sessionId, tabId);
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
					sessionId,
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

				// LoadGenerationSession now takes only sessionID
				const result = await LoadGenerationSession(sessionId);
				const restoredChat = normalizeChatMessages(
					(result as any)?.chatMessages
				);

				setDocState(sessionKey, {
					sessionId,
					projectId,
					projectName: finalProjectName,
					sessionKey,
					result,
					status: "success",
					sourceBranch: result?.branch ?? sourceBranch ?? null,
					targetBranch: (result?.targetBranch ?? targetBranch)?.trim() || null,
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
					messages: restoredChat,
					mergeInProgress: false,
				});
				setActiveSessionKey(projectKey, sessionKey);
				updateSessionMeta(sessionKey, {
					sessionId,
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
					sessionId,
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
			// Use temp session key since we don't have a session ID yet
			const sessionKey =
				sessionKeyOverride ??
				(tabId
					? createTempSessionKey(projectId, sourceBranch, tabId)
					: createTempSessionKey(projectId, sourceBranch));
			const resolvedTabId =
				tabId ?? findTabIdForSession(projectId, sessionKey) ?? undefined;
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

				// Delete any existing session with this docs branch first
				try {
					const existingSession = await GetByDocsBranch(projectId, existing);
					if (existingSession?.ID) {
						await DeleteGenerationSession(existingSession.ID);
					}
				} catch {
					// Ignore errors when looking up/deleting the session
					// The branch deletion and restart will still proceed
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
						tabId: resolvedTabId,
					});
				} else {
					await get().startFromBranch?.({
						projectId,
						projectName,
						sourceBranch,
						targetBranch: "",
						modelKey,
						userInstructions,
						tabId: resolvedTabId,
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
			// Use temp session key since we don't have a session ID yet
			const baseTempKey = createTempSessionKey(projectId, sourceBranch);
			const sessionKey =
				sessionKeyOverride ??
				(tabId
					? createTempSessionKey(projectId, sourceBranch, tabId)
					: baseTempKey);
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
				// Clear conflict immediately so the dialog closes
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
					docsBranch: targetName,
					mergeInProgress: false,
					initialDiffSignatures: null,
					changedSinceInitial: [],
					todos: [],
					conflict: null,
					events: [
						...prev.events,
						createLocalEvent(
							"info",
							`Using new docs branch name '${targetName}'`
						),
					],
				}));

				bindBackendSession(baseTempKey, sessionKey);
				subscribeToGenerationEvents(sessionKey, baseTempKey);

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

				// Extract sessionId from result and update state
				const newSessionId = result?.sessionId ?? null;

				// Success: populate result and status, clear conflict
				setDocState(sessionKey, (prev) => ({
					...prev,
					sessionId: newSessionId,
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
				updateSessionMeta(sessionKey, {
					sessionId: newSessionId,
					status: "success",
				});
			} catch (error) {
				// On failure, ensure any in-flight generation is stopped
				const docState = get().docStates[sessionKey];
				try {
					await StopStream(docState?.sessionId ?? 0, sessionKey);
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
				unbindBackendSession(baseTempKey, sessionKey);
				clearSubscriptions(sessionKey);
				setDocState(sessionKey, { cancellationRequested: false });
			}
		},
	};
});
