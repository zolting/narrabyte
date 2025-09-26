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
						label: key.label ?? "N/A",
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
			toast(t("apiDialog.keySaved"));
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
					<DialogTitle>{t("apiDialog.addApiKey")}</DialogTitle>
				</DialogHeader>

				{/* Existing keys list */}
				{existingKeys.length > 0 && (
					<div className="mb-4">
						<p className="text-sm font-medium">{t("apiDialog.existingKeys")}</p>
						<ul className="list-disc pl-4 text-sm text-gray-700 dark:text-gray-300">
							{existingKeys.map((k) => (
								<li key={k.provider}>{k.provider}</li>
							))}
						</ul>
					</div>
				)}

				<form className="space-y-4" onSubmit={handleSubmit}>
					<div className="space-y-2">
						<Label htmlFor={providerId}>Provider</Label>
						<Select value={provider} onValueChange={setProvider}>
							<SelectTrigger>
								<SelectValue placeholder="Select provider" />
							</SelectTrigger>
							<SelectContent>
								{PROVIDERS.map((p) => (
									<SelectItem key={p.name} value={p.name}>
										{p.key}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-2">
						<Label htmlFor={apiKeyId}>API Key</Label>
						<Input
							id={apiKeyId}
							onChange={(e) => setApiKey(e.target.value)}
							required
							type="text"
							value={apiKey}
						/>
					</div>
					<DialogFooter>
						<Button onClick={onClose} type="button" variant="outline">
							{t("common.cancel")}
						</Button>
						<Button type="submit">{"Save"}</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
