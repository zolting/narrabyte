import {
	DeleteApiKey,
	GetApiKey,
	ListApiKeys,
} from "@go/services/KeyringService";
import { Eye, EyeOff, Pencil, Plus, Trash2 } from "lucide-react";
import { forwardRef, useEffect, useImperativeHandle, useState } from "react";
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
}

const ApiKeyManager = forwardRef<ApiKeyManagerHandle, ApiKeyManagerProps>(
	({ onAddClick, onEditClick }, ref) => {
		const { t } = useTranslation();
		const [apiKeys, setApiKeys] = useState<ApiKeyInfo[]>([]);
		const [loading, setLoading] = useState(false);
		const [visibleKeys, setVisibleKeys] = useState<Record<string, string>>({});
		const [revealedKeys, setRevealedKeys] = useState<Set<string>>(new Set());

		// Check if all providers have keys
		const allProvidersHaveKeys =
			apiKeys.length > 0 &&
			PROVIDERS.every((p) => apiKeys.some((k) => k.provider === p.name));

		const loadApiKeys = async () => {
			setLoading(true);
			try {
				const keys = await ListApiKeys();
				// Handle both null/undefined and empty array cases
				if (!keys || keys.length === 0) {
					setApiKeys([]);
				} else {
					const mappedKeys: ApiKeyInfo[] = keys.map(
						(key: Record<string, string>) => ({
							provider: key.provider,
							label: key.label ?? "N/A",
							description: key.description,
						})
					);
					setApiKeys(mappedKeys);
				}
			} catch (err) {
				// Only show error toast if it's not a "no keys" situation
				console.error("Failed to fetch API keys:", err);
				// Set to empty array instead of showing error
				setApiKeys([]);
			} finally {
				setLoading(false);
			}
		};

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
				} catch (err) {
					console.error("Failed to fetch API key:", err);
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
			} catch (err) {
				console.error("Failed to delete API key:", err);
				toast.error(t("apiKeys.deleteError"));
			}
		};

		useEffect(() => {
			loadApiKeys();
		}, []);

		// Expose refresh method to parent via ref
		useImperativeHandle(ref, () => ({
			refresh: loadApiKeys,
		}));

		return (
			<Card>
				<CardHeader>
					<div className="flex items-center justify-between">
						<div>
							<CardTitle>{t("apiKeys.title")}</CardTitle>
							<CardDescription>{t("apiKeys.description")}</CardDescription>
						</div>
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
					</div>
				</CardHeader>
				<CardContent>
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
						<div className="space-y-2">
							{apiKeys.map((key) => (
								<div
									className="flex items-center justify-between gap-3 rounded-lg border p-3"
									key={key.provider}
								>
									<div className="min-w-0 flex-1">
										<div className="font-medium">{key.provider}</div>
										<div className="mt-1 truncate font-mono text-sm">
											{revealedKeys.has(key.provider)
												? visibleKeys[key.provider] || "***"
												: "••••••••••••••••"}
										</div>
									</div>
									<div className="flex gap-1">
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
			</Card>
		);
	}
);

ApiKeyManager.displayName = "ApiKeyManager";

export default ApiKeyManager;
