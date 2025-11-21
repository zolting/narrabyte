import {
	DeleteApiKey,
	GetApiKey,
	ListApiKeys,
} from "@go/services/KeyringService";
import { ChevronDown, Eye, EyeOff, Pencil, Plus, Trash2 } from "lucide-react";
import {
	forwardRef,
	useCallback,
	useEffect,
	useImperativeHandle,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { PROVIDERS } from "@/components/AddApiKeyDialog";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Collapsible,
	CollapsibleContent,
	CollapsibleTrigger,
} from "@/components/ui/collapsible";

type ApiKeyInfo = {
	provider: string;
	label: string;
	description: string;
};

export interface ApiKeyManagerHandle {
	refresh: () => void;
}

interface ApiKeyManagerProps {
	onAddClick: () => void;
	onEditClick: (provider: string) => void;
	onKeysChanged?: () => void;
}

const ApiKeyManager = forwardRef<ApiKeyManagerHandle, ApiKeyManagerProps>(
	({ onAddClick, onEditClick, onKeysChanged }, ref) => {
		const { t } = useTranslation();
		const [apiKeys, setApiKeys] = useState<ApiKeyInfo[]>([]);
		const [loading, setLoading] = useState(false);
		const [visibleKeys, setVisibleKeys] = useState<Record<string, string>>({});
		const [revealedKeys, setRevealedKeys] = useState<Set<string>>(new Set());
		// Collapsed by default if there are API keys, expanded if empty
		const [isOpen, setIsOpen] = useState(false);

		// Check if all providers have keys
		const allProvidersHaveKeys =
			apiKeys.length > 0 &&
			PROVIDERS.every((p) => apiKeys.some((k) => k.provider === p.name));

		const loadApiKeys = useCallback(async () => {
			setLoading(true);
			try {
				const keys = await ListApiKeys();
				// Handle both null/undefined and empty array cases
				if (!keys || keys.length === 0) {
					setApiKeys([]);
					// Open if there are no keys
					setIsOpen(true);
				} else {
					const mappedKeys: ApiKeyInfo[] = keys.map(
						(key: Record<string, string>) => ({
							provider: key.provider,
							label: key.label ?? "N/A",
							description: key.description,
						})
					);
					setApiKeys(mappedKeys);
					// Keep current state or close if this is the first load
					setIsOpen((prev) => (prev ? prev : false));
				}
			} catch (_err) {
				// Set to empty array instead of showing error
				setApiKeys([]);
				setIsOpen(true);
			} finally {
				setLoading(false);
			}
		}, []);

		const toggleKeyVisibility = async (provider: string) => {
			if (revealedKeys.has(provider)) {
				// Hide the key
				setRevealedKeys((prev) => {
					const next = new Set(prev);
					next.delete(provider);
					return next;
				});
				setVisibleKeys((prev) => {
					const next = { ...prev };
					delete next[provider];
					return next;
				});
			} else {
				// Fetch and show the key
				try {
					const apiKey = await GetApiKey(provider);
					setVisibleKeys((prev) => ({ ...prev, [provider]: apiKey }));
					setRevealedKeys((prev) => new Set(prev).add(provider));
				} catch (_err) {
					toast.error(t("apiKeys.fetchError"));
				}
			}
		};

		const handleDelete = async (provider: string) => {
			try {
				await DeleteApiKey(provider);
				// Clean up revealed state
				setRevealedKeys((prev) => {
					const next = new Set(prev);
					next.delete(provider);
					return next;
				});
				setVisibleKeys((prev) => {
					const next = { ...prev };
					delete next[provider];
					return next;
				});
				// Update the list immediately without refetching
				setApiKeys((prev) => prev.filter((key) => key.provider !== provider));
				toast.success(t("apiKeys.deleteSuccess"));
				// Notify parent that keys changed so other UI (models list) can refresh
				try {
					onKeysChanged?.();
				} catch {
					// ignore
				}
			} catch (_err) {
				toast.error(t("apiKeys.deleteError"));
			}
		};

		useEffect(() => {
			loadApiKeys();
		}, [loadApiKeys]);

		// Expose refresh method to parent via ref
		useImperativeHandle(ref, () => ({
			refresh: loadApiKeys,
		}));

		return (
			<Collapsible onOpenChange={setIsOpen} open={isOpen}>
				<Card className="overflow-hidden">
					<CardHeader>
						<div className="flex items-center justify-between">
							<div className="flex-1">
								<CardTitle>{t("apiKeys.title")}</CardTitle>
								<CardDescription>{t("apiKeys.description")}</CardDescription>
							</div>
							<div className="flex shrink-0 items-center gap-2">
								<Button
									disabled={allProvidersHaveKeys}
									onClick={onAddClick}
									size="sm"
									title={
										allProvidersHaveKeys
											? t("apiKeys.allProvidersConfigured")
											: undefined
									}
								>
									<Plus className="mr-1 h-4 w-4" />
									{t("apiKeys.add")}
								</Button>
								<CollapsibleTrigger asChild>
									<Button
										aria-label={
											isOpen ? "Collapse API Keys" : "Expand API Keys"
										}
										size="sm"
										variant="ghost"
									>
										<ChevronDown
											className={`h-4 w-4 transition-transform ${isOpen ? "rotate-180" : ""}`}
										/>
									</Button>
								</CollapsibleTrigger>
							</div>
						</div>
					</CardHeader>
					<CollapsibleContent>
						<CardContent className="overflow-hidden">
							{loading && (
								<div className="text-center text-muted-foreground text-sm">
									{t("settings.loading")}
								</div>
							)}
							{!loading && apiKeys.length === 0 && (
								<div className="text-center text-muted-foreground text-sm">
									{t("apiKeys.noKeys")}
								</div>
							)}
							{!loading && apiKeys.length > 0 && (
								<div className="space-y-2 overflow-hidden">
									{apiKeys.map((key) => (
										<div
											className="flex items-center gap-3 overflow-hidden rounded-lg border p-3"
											key={key.provider}
										>
											<div className="min-w-0 flex-1">
												<div className="font-medium">{key.provider}</div>
												<div className="mt-1 overflow-hidden text-ellipsis whitespace-nowrap font-mono text-sm">
													{revealedKeys.has(key.provider)
														? visibleKeys[key.provider] || "***"
														: "••••••••••••••••"}
												</div>
											</div>
											<div className="flex shrink-0 gap-1">
												<Button
													onClick={() => toggleKeyVisibility(key.provider)}
													size="sm"
													title={
														revealedKeys.has(key.provider)
															? t("apiKeys.hideKey")
															: t("apiKeys.showKey")
													}
													variant="ghost"
												>
													{revealedKeys.has(key.provider) ? (
														<EyeOff className="h-4 w-4" />
													) : (
														<Eye className="h-4 w-4" />
													)}
												</Button>
												<Button
													onClick={() => onEditClick(key.provider)}
													size="sm"
													title={t("apiKeys.editKey")}
													variant="ghost"
												>
													<Pencil className="h-4 w-4" />
												</Button>
												<Button
													onClick={() => handleDelete(key.provider)}
													size="sm"
													title={t("apiKeys.deleteKey")}
													variant="ghost"
												>
													<Trash2 className="h-4 w-4" />
												</Button>
											</div>
										</div>
									))}
								</div>
							)}
						</CardContent>
					</CollapsibleContent>
				</Card>
			</Collapsible>
		);
	}
);

ApiKeyManager.displayName = "ApiKeyManager";

export default ApiKeyManager;
