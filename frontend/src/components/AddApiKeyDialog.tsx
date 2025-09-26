import { useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
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
	ListApiKeys,
	StoreApiKey,
} from "../../wailsjs/go/services/KeyringService";

const PROVIDERS = [
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
}: {
	open: boolean;
	onClose: () => void;
}) {
	const providerId = useId();
	const apiKeyId = useId();
	const [provider, setProvider] = useState(PROVIDERS[0].name);
	const [apiKey, setApiKey] = useState("");
	const [existingKeys, setExistingKeys] = useState<ApiKeyInfo[]>([]);
	const { t } = useTranslation();

	useEffect(() => {
		if (open) {
			ListApiKeys()
				.then((keys: Record<string, string>[]) => {
					// Map each object to ApiKeyInfo
					const apiKeys: ApiKeyInfo[] = keys.map((key) => ({
						provider: key.provider,
						label: key.label ?? "foo",
						description: key.description,
					}));
					setExistingKeys(apiKeys);
				})
				.catch((err: Error) => console.error("Failed to fetch API keys:", err));
		}
	}, [open]);

	const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		try {
			//API key needs to be converted to byte array
			const apiKeyBytes = stringToByteArray(apiKey);
			await StoreApiKey(provider, apiKeyBytes);
			alert(t("apiDialog.keySaved"));
			onClose();
		} catch (error) {
			alert(t("apiDialog.errSavingKey") + error);
		}
	};

	if (!open) return null;

	return (
		<Dialog open={open} onOpenChange={onClose}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{t("apiDialog.addApiKey")}</DialogTitle>
				</DialogHeader>

				{/* Existing keys list */}
				{existingKeys.length > 0 && (
					<div className="mb-4">
						<p className="text-sm font-medium">{t("apiDialog.existingKeys")}</p>
						<ul className="list-disc pl-4 text-sm text-muted-foreground">
							{existingKeys.map((k) => (
								<li key={k.provider}>{k.provider}</li>
							))}
						</ul>
					</div>
				)}

				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-2">
						<Label htmlFor={providerId}>Provider</Label>
						<select
							id={providerId}
							className="w-full rounded-md border px-3 py-2"
							value={provider}
							onChange={(e) => setProvider(e.target.value)}
						>
							{PROVIDERS.map((p) => (
								<option key={p.name} value={p.name}>
									{p.key}
								</option>
							))}
						</select>
					</div>
					<div className="space-y-2">
						<Label htmlFor={apiKeyId}>API Key</Label>
						<Input
							id={apiKeyId}
							type="text"
							value={apiKey}
							onChange={(e) => setApiKey(e.target.value)}
							required
						/>
					</div>
					<DialogFooter>
						<Button type="button" variant="outline" onClick={onClose}>
							{t("common.cancel")}
						</Button>
						<Button type="submit">{"Save"}</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
