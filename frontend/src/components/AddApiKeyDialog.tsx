import { ListApiKeys, StoreApiKey } from "@go/services/KeyringService";
import { useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";

export const PROVIDERS = [
	{ name: "openai", key: "OpenAI" },
	{ name: "openrouter", key: "OpenRouter" },
];

// Convert string to []byte (UTF-8 encoding)
function stringToByteArray(str: string): number[] {
	return Array.from(new TextEncoder().encode(str));
}

//Used to list the active API keys. is it unsafe tho..?
type ApiKeyInfo = {
	provider: string;
	label: string;
	description: string;
};

export default function AddApiKeyDialog({
	open,
	onClose,
	onKeyAdded,
	editProvider,
}: {
	open: boolean;
	onClose: () => void;
	onKeyAdded?: () => void;
	editProvider?: string;
}) {
	const providerId = useId();
	const apiKeyId = useId();
	const [provider, setProvider] = useState(editProvider || PROVIDERS[0].name);
	const [apiKey, setApiKey] = useState("");
	const [existingKeys, setExistingKeys] = useState<ApiKeyInfo[]>([]);
	const { t } = useTranslation();
	const isEditing = !!editProvider;

	useEffect(() => {
		if (open) {
			// Load existing keys
			ListApiKeys()
				.then((keys: Record<string, string>[]) => {
					// Map each object to ApiKeyInfo
					const apiKeys: ApiKeyInfo[] = keys.map((key) => ({
						provider: key.provider,
						label: key.label ?? "N/A",
						description: key.description,
					}));
					setExistingKeys(apiKeys);

					// Set provider if editing
					if (editProvider) {
						setProvider(editProvider);
					} else {
						// Find first available provider that doesn't have a key
						const existingProviders = new Set(apiKeys.map((k) => k.provider));
						const availableProvider = PROVIDERS.find(
							(p) => !existingProviders.has(p.name)
						);
						if (availableProvider) {
							setProvider(availableProvider.name);
						}
					}
				})
				.catch((err: Error) => console.error("Failed to fetch API keys:", err));
		}
	}, [open, editProvider]);

	const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		try {
			//API key needs to be converted to byte array
			const apiKeyBytes = stringToByteArray(apiKey);
			await StoreApiKey(provider, apiKeyBytes);
			toast(isEditing ? t("apiDialog.keyUpdated") : t("apiDialog.keySaved"));
			setApiKey(""); // Clear the input
			onKeyAdded?.(); // Notify parent to refresh
			onClose();
		} catch (error) {
			toast(t("apiDialog.errSavingKey") + error);
		}
	};

	if (!open) {
		return null;
	}

	return (
		<Dialog onOpenChange={onClose} open={open}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>
						{isEditing ? t("apiDialog.editApiKey") : t("apiDialog.addApiKey")}
					</DialogTitle>
				</DialogHeader>

				<form className="space-y-4" onSubmit={handleSubmit}>
					<div className="space-y-2">
						<Label htmlFor={providerId}>Provider</Label>
						<Select
							disabled={isEditing}
							onValueChange={setProvider}
							value={provider}
						>
							<SelectTrigger>
								<SelectValue placeholder="Select provider" />
							</SelectTrigger>
							<SelectContent>
								{PROVIDERS.filter((p) => {
									// When editing, only show the current provider
									if (isEditing) {
										return p.name === editProvider;
									}
									// When adding, only show providers that don't have keys yet
									return !existingKeys.some((k) => k.provider === p.name);
								}).map((p) => (
									<SelectItem key={p.name} value={p.name}>
										{p.key}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-2">
						<Label htmlFor={apiKeyId}>
							{isEditing ? t("apiDialog.newApiKey") : "API Key"}
						</Label>
						<Input
							id={apiKeyId}
							onChange={(e) => setApiKey(e.target.value)}
							placeholder={isEditing ? t("apiDialog.enterNewKey") : undefined}
							required
							type="text"
							value={apiKey}
						/>
					</div>
					<DialogFooter>
						<Button onClick={onClose} type="button" variant="outline">
							{t("common.cancel")}
						</Button>
						<Button type="submit">
							{isEditing ? t("apiDialog.update") : t("apiDialog.save")}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
