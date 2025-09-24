import type React from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { StoreApiKey } from "../../wailsjs/go/services/KeyringService";

const PROVIDERS = [
	{ name: "openai", key: "OpenAI" },
	{ name: "openrouter", key: "OpenRouter" },
];

// Convert string to []byte (UTF-8 encoding)
function stringToByteArray(str: string): number[] {
	return Array.from(new TextEncoder().encode(str));
}

export default function AddApiKeyDialog({
	open,
	onClose,
}: {
	open: boolean;
	onClose: () => void;
}) {
	const [provider, setProvider] = useState(PROVIDERS[0].name);
	const [apiKey, setApiKey] = useState("");
	const { t } = useTranslation();

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
		<div className="dialog-backdrop">
			<div className="dialog">
				<h2>Add an API key</h2>
				<form onSubmit={handleSubmit}>
					<label>
						Provider:
						<select
							value={provider}
							onChange={(e) => setProvider(e.target.value)}
						>
							{PROVIDERS.map((p) => (
								<option key={p.name} value={p.name}>
									{p.key}
								</option>
							))}
						</select>
					</label>
					<label>
						API Key:
						<input
							type="text"
							value={apiKey}
							onChange={(e) => setApiKey(e.target.value)}
							required
						/>
					</label>
					<div className="dialog-actions">
						<button type="button" onClick={onClose}>
							Cancel
						</button>
						<button type="submit">Save</button>
					</div>
				</form>
			</div>
		</div>
	);
}
