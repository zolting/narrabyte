import type { models } from "@go/models";
import { create } from "zustand";
import { parseDiff } from "react-diff-view";
import { type DemoEvent, demoEventSchema } from "@/types/events";
import {
	CommitDocs,
	GenerateDocs,
	RefineDocs,
	StopStream,
} from "../../wailsjs/go/services/ClientService";
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
	sourceBranch: string;
	targetBranch: string;
	provider: string;
};

type CommitArgs = {
	projectId: number;
	branch: string;
	files: string[];
};

type ProjectKey = string;

type CompletedCommitInfo = {
	sourceBranch: string;
	targetBranch: string;
};

type ChatMessage = {
	id: string;
	role: "user" | "assistant";
	content: string;
	status?: "pending" | "sent" | "error";
	createdAt: Date;
};

type DocGenerationData = {
	events: DemoEvent[];
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
};

type State = {
	docStates: Record<ProjectKey, DocGenerationData>;
	start: (args: StartArgs) => Promise<void>;
	reset: (projectId: number | string) => void;
	commit: (args: CommitArgs) => Promise<void>;
	cancel: (projectId: number | string) => Promise<void>;
	setActiveTab: (projectId: number | string, tab: "activity" | "review" | "summary") => void;
	setCommitCompleted: (projectId: number | string, completed: boolean) => void;
	setCompletedCommitInfo: (
		projectId: number | string,
		info: CompletedCommitInfo | null
	) => void;
	toggleChat: (projectId: number | string, open?: boolean) => void;
	refine: (args: { projectId: number; branch: string; instruction: string }) => Promise<void>;
};

const EMPTY_DOC_STATE: DocGenerationData = {
	events: [],
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
};

const toKey = (projectId: number | string): ProjectKey => String(projectId);

const messageFromError = (error: unknown) => {
	if (error instanceof Error) {
		return error.message;
	}
	if (typeof error === "string") {
		return error;
	}
	return "An unknown error occurred while generating documentation.";
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
	if (!diffText || !diffText.trim()) {
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
};

const subscriptions = new Map<ProjectKey, SubscriptionMap>();

const clearSubscriptions = (key: ProjectKey) => {
	const entry = subscriptions.get(key);
	if (!entry) {
		return;
	}
	entry.tool?.();
	entry.done?.();
	subscriptions.delete(key);
};

export const useDocGenerationStore = create<State>((set, get) => {
	const setDocState = (
		key: ProjectKey,
		partial: Partial<DocGenerationData> | ((prev: DocGenerationData) => DocGenerationData)
	) => {
		set((state) => {
			const previous = state.docStates[key] ?? EMPTY_DOC_STATE;
			const next =
				typeof partial === "function" ? partial(previous) : { ...previous, ...partial };
			return {
				docStates: {
					...state.docStates,
					[key]: next,
				},
			};
		});
	};

	return {
		docStates: {},

		start: async ({
			projectId,
			sourceBranch,
			targetBranch,
			provider,
		}: StartArgs) => {
			const key = toKey(projectId);
			const currentState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (currentState.status === "running") {
				return;
			}

			setDocState(key, {
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
			});

			clearSubscriptions(key);

			const toolUnsub = EventsOn("event:llm:tool", (payload) => {
				try {
					const evt = demoEventSchema.parse(payload);
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
					setDocState(key, (prev) => ({
						...prev,
						events: [...prev.events, evt],
					}));
				} catch (error) {
					console.error("Invalid doc generation done event", error, payload);
				}
			});

			subscriptions.set(key, { tool: toolUnsub, done: doneUnsub });

			try {
				const result = await GenerateDocs(
					projectId,
					sourceBranch,
					targetBranch,
					provider
				);
				setDocState(key, {
					result,
					status: "success",
					cancellationRequested: false,
					initialDiffSignatures: computeDiffSignatures(result?.diff ?? null),
					changedSinceInitial: [],
				});
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
			} finally {
				clearSubscriptions(key);
				setDocState(key, { cancellationRequested: false });
			}
		},

		cancel: async (projectId: number | string) => {
			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (docState.status !== "running") {
				return;
			}

			setDocState(key, { cancellationRequested: true });
			try {
				await StopStream();
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
			}
		},

		commit: async ({ projectId, branch, files }: CommitArgs) => {
			const key = toKey(projectId);
			const docState = get().docStates[key] ?? EMPTY_DOC_STATE;
			if (docState.status === "committing") {
				return;
			}

			setDocState(key, (prev) => ({
				...prev,
				error: null,
				status: "committing",
				events: [
					...prev.events,
					createLocalEvent(
						"info",
						`Committing documentation updates to ${branch}`
					),
				],
				activeTab: "activity",
				commitCompleted: false,
			}));

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
							`Committed documentation changes for ${branch}`
						),
					],
					commitCompleted: true,
				}));
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
			}
		},

		reset: (projectId: number | string) => {
			const key = toKey(projectId);
			clearSubscriptions(key);
			setDocState(key, {
				events: [],
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
				status: "pending",
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
						prev.initialDiffSignatures ?? computeDiffSignatures(prev.result?.diff ?? null);
					const current = computeDiffSignatures(result?.diff ?? null);
					const changed = Object.keys(current).filter((path) => {
						const previousSignature = baseline[path];
						if (previousSignature === undefined) {
							return true;
						}
						return previousSignature !== current[path];
					});
					const updatedMessages = prev.messages.map((message) =>
						message.id === messageId ? { ...message, status: "sent" } : message
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
					};
				});
			} catch (error) {
				const message = messageFromError(error);
				setDocState(key, (prev) => {
					const updatedMessages = prev.messages.map((msg) =>
						msg.id === messageId ? { ...msg, status: "error" } : msg
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
	};
});
