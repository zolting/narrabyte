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
import i18n from "i18next";
import { parseDiff } from "react-diff-view";
import { create } from "zustand";
import {
	type DemoEvent,
	demoEventSchema,
	type TodoItem,
	todoEventSchema,
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
type SessionKey = string;

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

type DocGenerationData = {
	projectId: number;
	projectName: string;
	sessionKey: SessionKey;
	events: DemoEvent[];
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
};

type State = {
	docStates: Record<ProjectKey, DocGenerationData>;
	sessionMeta: Record<SessionKey, SessionMeta>;
	activeSession: Record<ProjectKey, SessionKey | null>;
	start: (args: StartArgs) => Promise<void>;
	startFromBranch?: (args: StartArgs) => Promise<void>;
	reset: (projectId: number | string) => void;
	commit: (args: CommitArgs) => Promise<void>;
	cancel: (projectId: number | string) => Promise<void>;
	setActiveTab: (
		projectId: number | string,
		tab: "activity" | "review" | "summary"
	) => void;
	setCommitCompleted: (projectId: number | string, completed: boolean) => void;
	setCompletedCommitInfo: (
		projectId: number | string,
		info: CompletedCommitInfo | null
	) => void;
	toggleChat: (projectId: number | string, open?: boolean) => void;
	refine: (args: {
		projectId: number;
		branch: string;
		instruction: string;
	}) => Promise<void>;
	mergeDocs: (args: { projectId: number; branch: string }) => Promise<void>;
	restoreSession: (
		projectId: number,
		sourceBranch: string,
		targetBranch: string
	) => Promise<boolean>;
	setActiveSession: (
		projectId: number | string,
		sessionKey: SessionKey | null
	) => void;
	clearSessionMeta: (projectId: number, sourceBranch: string) => void;
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
const createSessionKey = (
	projectId: number,
	sourceBranch: string | null | undefined
): SessionKey => `${projectId}:${(sourceBranch ?? "").trim()}`;

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

	// Check for specific error codes
	if (trimmed === "ERR_UNCOMMITTED_CHANGES_ON_SOURCE_BRANCH") {
		return i18n.t(
			"common.mergeDisabledUncommittedChanges",
			"Cannot merge: You are currently on the source branch with uncommitted changes. Please commit or stash your changes first."
		);
	}

	// Return original message if no mapping found
	return errorMessage;
};

const createLocalEvent = (
	type: DemoEvent["type"],
	message: string
): DemoEvent => ({
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

const subscriptions = new Map<ProjectKey, SubscriptionMap>();

const clearSubscriptions = (key: ProjectKey) => {
	const entry = subscriptions.get(key);
	if (!entry) {
		return;
	}
	entry.tool?.();
	entry.done?.();
	entry.todo?.();
	subscriptions.delete(key);
};

export const useDocGenerationStore = create<State>((set, get) => {
	const setDocState = (
		key: ProjectKey,
		partial:
			| Partial<DocGenerationData>
			| ((prev: DocGenerationData) => DocGenerationData)
	) => {
		set((state) => {
			const previous = state.docStates[key] ?? EMPTY_DOC_STATE;
			const next =
				typeof partial === "function"
					? partial(previous)
					: { ...previous, ...partial };
			return {
				docStates: {
					...state.docStates,
					[key]: next,
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

	return {
		docStates: {},
		sessionMeta: {},
		activeSession: {},

		start: async ({
			projectId,
			projectName,
			sourceBranch,
			targetBranch,
			modelKey,
			userInstructions,
		}: StartArgs) => {
			const key = toKey(projectId);
			const sessionKey = createSessionKey(projectId, sourceBranch);
			const currentState = get().docStates[key];
			if (currentState?.status === "running") {
				return;
			}

			setDocState(key, {
				projectId,
				projectName,
				sessionKey,
				events: [],
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
			setActiveSessionKey(key, sessionKey);
			updateSessionMeta(sessionKey, {
				projectId,
				projectName,
				sourceBranch,
				targetBranch,
				status: "running",
			});

			clearSubscriptions(key);

			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					if (evt.sessionKey && evt.sessionKey !== sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation tool event", error, payload);
				}
			});

			const doneUnsub = EventsOn("events:llm:done", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					if (evt.sessionKey && evt.sessionKey !== sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
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
					if (evt.sessionKey && evt.sessionKey !== sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
						...prev,
						todos: evt.todos,
					}));
				} catch (error) {
					console.error("Invalid todo event", error, payload);
				}
			});

			subscriptions.set(key, {
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
					userInstructions
				);
				setDocState(key, {
					result,
					status: "success",
					cancellationRequested: false,
					initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
					changedSinceInitial: [],
					docsInCodeRepo: Boolean(result?.docsInCodeRepo),
					docsBranch: result?.docsBranch ?? null,
					mergeInProgress: false,
				});
				removeSessionMeta(sessionKey);
			} catch (error) {
				const message = messageFromError(error);
				const normalized = message.toLowerCase();
				const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
				const canceled =
					docState.cancellationRequested ||
					normalized.includes("context canceled") ||
					normalized.includes("context cancelled") ||
					normalized.includes("cancelled") ||
					normalized.includes("canceled");
				if (canceled) {
					setDocState(key, (prev) => ({
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
				} else {
					setDocState(key, {
						error: message,
						status: "error",
						cancellationRequested: false,
						result: null,
						commitCompleted: false,
					});
				}
				removeSessionMeta(sessionKey);
			} finally {
				clearSubscriptions(key);
				setDocState(key, { cancellationRequested: false });
			}
		},

		startFromBranch: async ({
			projectId,
			projectName,
			sourceBranch,
			modelKey,
			userInstructions,
		}: StartArgs) => {
			const key = toKey(projectId);
			const sessionKey = createSessionKey(projectId, sourceBranch);
			const currentState = get().docStates[key];
			if (currentState?.status === "running") {
				return;
			}

			setDocState(key, {
				projectId,
				projectName,
				sessionKey,
				events: [],
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
			setActiveSessionKey(key, sessionKey);
			updateSessionMeta(sessionKey, {
				projectId,
				projectName,
				sourceBranch,
				targetBranch: "",
				status: "running",
			});

			clearSubscriptions(key);

			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					if (evt.sessionKey && evt.sessionKey !== sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation tool event", error, payload);
				}
			});

			const doneUnsub = EventsOn("events:llm:done", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					if (evt.sessionKey && evt.sessionKey !== sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
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
					if (evt.sessionKey && evt.sessionKey !== sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
						...prev,
						todos: evt.todos,
					}));
				} catch (error) {
					console.error("Invalid todo event", error, payload);
				}
			});

			subscriptions.set(key, {
				tool: toolUnsub,
				done: doneUnsub,
				todo: todoUnsub,
			});

			try {
				const result = await GenerateDocsFromBranch(
					projectId,
					sourceBranch,
					modelKey,
					userInstructions
				);
				setDocState(key, {
					result,
					status: "success",
					cancellationRequested: false,
					initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
					changedSinceInitial: [],
					docsInCodeRepo: Boolean(result?.docsInCodeRepo),
					docsBranch: result?.docsBranch ?? null,
					mergeInProgress: false,
				});
				removeSessionMeta(sessionKey);
			} catch (error) {
				const message = messageFromError(error);
				const normalized = message.toLowerCase();
				const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
				const canceled =
					docState.cancellationRequested ||
					normalized.includes("context canceled") ||
					normalized.includes("context cancelled") ||
					normalized.includes("cancelled") ||
					normalized.includes("canceled");
				if (canceled) {
					setDocState(key, (prev) => ({
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
				} else {
					setDocState(key, {
						error: message,
						status: "error",
						cancellationRequested: false,
						result: null,
						commitCompleted: false,
					});
				}
				removeSessionMeta(sessionKey);
			} finally {
				clearSubscriptions(key);
				setDocState(key, { cancellationRequested: false });
			}
		},

		cancel: async (
			projectId: number | string,
			sourceBranch?: string | null
		) => {
			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (docState.status !== "running") {
				return;
			}

			const branch = sourceBranch ?? docState.sourceBranch ?? "";
			const sessionKey = docState.sessionKey;
			setDocState(key, { cancellationRequested: true });
			try {
				await StopStream(Number(projectId), branch);
				removeSessionMeta(sessionKey);
			} catch (error) {
				const message = messageFromError(error);
				console.error("Failed to cancel doc generation", error);
				setDocState(key, (prev) => ({
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
				removeSessionMeta(sessionKey);
			}
		},

		commit: async ({ projectId, branch, files }: CommitArgs) => {
			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (docState.status === "committing") {
				return;
			}
			const sessionKey = docState.sessionKey;

			const label =
				docState.docsBranch && docState.docsBranch.trim() !== ""
					? docState.docsBranch
					: branch;

			setDocState(key, (prev) => ({
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
			if (sessionKey) {
				updateSessionMeta(sessionKey, (prev) => ({
					projectId: prev?.projectId ?? docState.projectId,
					projectName: prev?.projectName ?? docState.projectName,
					sourceBranch: prev?.sourceBranch ?? docState.sourceBranch ?? "",
					targetBranch: prev?.targetBranch ?? docState.targetBranch ?? "",
					status: "committing",
				}));
			}

			try {
				await CommitDocs(projectId, branch, files);
				setDocState(key, (prev) => ({
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
				if (sessionKey) {
					removeSessionMeta(sessionKey);
				}
			} catch (error) {
				const message = messageFromError(error);
				setDocState(key, (prev) => ({
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
				if (sessionKey) {
					removeSessionMeta(sessionKey);
				}
			}
		},

		setActiveSession: (projectId, sessionKey) => {
			setActiveSessionKey(toKey(projectId), sessionKey);
		},

		clearSessionMeta: (projectId, sourceBranch) => {
			removeSessionMeta(createSessionKey(projectId, sourceBranch));
		},

		mergeDocs: async ({ projectId, branch }): Promise<void> => {
			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (!docState.docsInCodeRepo || docState.mergeInProgress) {
				return;
			}
			setDocState(key, (prev) => ({
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
				setDocState(key, (prev) => ({
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
				setDocState(key, (prev) => ({
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

		reset: (projectId: number | string) => {
			const key = toKey(projectId);
			clearSubscriptions(key);
			const current = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (current.sessionKey) {
				removeSessionMeta(current.sessionKey);
			}
			setDocState(key, {
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

		setActiveTab: (projectId, tab) => {
			const key = toKey(projectId);
			setDocState(key, { activeTab: tab });
		},

		setCommitCompleted: (projectId, completed) => {
			const key = toKey(projectId);
			setDocState(key, { commitCompleted: completed });
		},

		setCompletedCommitInfo: (projectId, info) => {
			const key = toKey(projectId);
			setDocState(key, { completedCommitInfo: info });
		},

		toggleChat: (projectId, open) => {
			const key = toKey(projectId);
			setDocState(key, (prev) => ({
				...prev,
				chatOpen: typeof open === "boolean" ? open : !prev.chatOpen,
			}));
		},

		refine: async ({ projectId, branch, instruction }) => {
			const trimmed = instruction.trim();
			if (!trimmed) {
				return;
			}

			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (docState.status === "running") {
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

			setDocState(key, (prev) => ({
				...prev,
				messages: [...prev.messages, userMessage],
				error: null,
				status: "running",
			}));

			clearSubscriptions(key);
			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					if (evt.sessionKey && evt.sessionKey !== docState.sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid refine tool event", error, payload);
				}
			});
			const doneUnsub = EventsOn("events:llm:done", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
					if (evt.sessionKey && evt.sessionKey !== docState.sessionKey) {
						return;
					}
					setDocState(key, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid refine done event", error, payload);
				}
			});

			subscriptions.set(key, { tool: toolUnsub, done: doneUnsub });

			try {
				const result = await RefineDocs(projectId, branch, trimmed);
				setDocState(key, (prev) => {
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
				setDocState(key, (prev) => {
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
				clearSubscriptions(key);
				setDocState(key, { cancellationRequested: false });
			}
		},

		restoreSession: async (
			projectId: number,
			sourceBranch: string,
			targetBranch: string
		): Promise<boolean> => {
			const key = toKey(projectId);
			const currentState = get().docStates[key];
			const sessionKey = createSessionKey(projectId, sourceBranch);

			if (
				currentState &&
				(currentState.status !== "idle" || currentState.result)
			) {
				return false;
			}

			try {
				const sessionMeta = get().sessionMeta[sessionKey];
				const projectName =
					currentState?.projectName ?? sessionMeta?.projectName ?? "";
				setDocState(key, {
					projectId,
					projectName,
					sessionKey,
					status: "running",
					events: [
						createLocalEvent(
							"info",
							`Restoring documentation session for branches: ${sourceBranch} → ${targetBranch}`
						),
					],
				});

				const result = await LoadGenerationSession(
					projectId,
					sourceBranch,
					targetBranch
				);

				setDocState(key, {
					projectId,
					projectName,
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
				});
				setActiveSessionKey(key, sessionKey);
				return true;
			} catch (error) {
				const message = messageFromError(error);
				console.error("Failed to restore generation session", error);
				setDocState(key, {
					status: "idle",
					events: [
						createLocalEvent("warn", `Could not restore session: ${message}`),
					],
				});
				return false;
			}
		},
	};
});
