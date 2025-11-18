import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useModelSettingsStore } from "@/stores/modelSettings";

type DefaultModelProps = {
	// Optional controlled value. If provided, Select uses this key.
	defaultModelKey?: string;
	// Optional change handler. If provided, called when user selects a model.
	onChange?: (key: string) => void;
	// Called when user confirms selection. Use this to persist the default model.
	onConfirm?: (key: string) => Promise<void> | void;
};

export default function DefaultModel(props: DefaultModelProps) {
	const { t } = useTranslation();
	const { defaultModelKey, onChange, onConfirm } = props;

	// Pull models state from the store
	const { groups, initialized, loading, error, init } = useModelSettingsStore();

	// Init models once if not already initialized
	useEffect(() => {
		if (!initialized) {
			init();
		}
	}, [init, initialized]);

	const allModels = useMemo(() => groups.flatMap((g) => g.models), [groups]);
	const enabledModels = useMemo(
		() => allModels.filter((model) => model.enabled),
		[allModels]
	);

	// Uncontrolled local selection fallback (when no defaultModelKey/onChange provided)
	const [localKey, setLocalKey] = useState<string>("");

	// Track last confirmed selection. Null = never confirmed => button active by default.
	const [confirmedKey, setConfirmedKey] = useState<string | null>(null);

	// Saving state for confirm action
	const [saving, setSaving] = useState(false);

	useEffect(() => {
		setConfirmedKey(
			defaultModelKey && defaultModelKey.length > 0 ? defaultModelKey : null
		);
	}, [defaultModelKey]);

	useEffect(() => {
		if (localKey && !enabledModels.some((model) => model.key === localKey)) {
			setLocalKey("");
		}
	}, [enabledModels, localKey]);

	// Compute the current value for the Select:
	// Prefer a user's in-progress local selection (unconfirmed) if present,
	// otherwise fall back to the controlled prop from settings, and finally
	// the first available model.
	const effectiveValue = useMemo(() => {
		if (localKey && enabledModels.some((model) => model.key === localKey)) {
			return localKey;
		}
		if (
			defaultModelKey &&
			enabledModels.some((model) => model.key === defaultModelKey)
		) {
			return defaultModelKey;
		}
		return "";
	}, [defaultModelKey, enabledModels, localKey]);

	const handleSelect = (value: string) => {
		if (onChange) {
			onChange(value);
			return;
		}
		setLocalKey(value);
	};

	const handleConfirm = async () => {
		if (!effectiveValue || saving) {
			return;
		}
		try {
			setSaving(true);
			await onConfirm?.(effectiveValue);
			// After confirming, disable the button until the selection changes
			setConfirmedKey(effectiveValue);
			// If uncontrolled, sync local as "confirmed"
			if (!defaultModelKey) {
				setLocalKey(effectiveValue);
			}
		} finally {
			setSaving(false);
		}
	};

	// Button should be enabled if:
	// - we never confirmed yet (confirmedKey === null), or
	// - the current selection differs from the last confirmed one
	const isDirty = confirmedKey === null || effectiveValue !== confirmedKey;

	const confirmDisabled =
		saving || !effectiveValue || enabledModels.length === 0 || !isDirty;

	return (
		<Card>
			<CardHeader>
				<CardTitle>{t("models.defaultModelTitle")}</CardTitle>
				<CardDescription>{t("models.defaultModelDescription")}</CardDescription>
			</CardHeader>
			<CardContent className="space-y-4">
				{error && (
					<p className="text-destructive text-sm" role="alert">
						{t("models.error")}
					</p>
				)}
				{loading && !initialized && (
					<p className="text-muted-foreground text-sm">{t("models.loading")}</p>
				)}
				{!loading && initialized && allModels.length === 0 && (
					<p className="text-muted-foreground text-sm">{t("models.empty")}</p>
				)}
				{!loading &&
					initialized &&
					allModels.length > 0 &&
					enabledModels.length === 0 && (
						<p className="text-muted-foreground text-sm">
							{t("models.noEnabledModels")}
						</p>
					)}

				<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
					<Select
						disabled={loading || enabledModels.length === 0}
						onValueChange={handleSelect}
						value={effectiveValue}
					>
						<SelectTrigger className="w-full sm:w-[280px]">
							<SelectValue
								placeholder={t("models.defaultModelSelectPlaceholder")}
							/>
						</SelectTrigger>
						<SelectContent>
							{enabledModels.map((m) => (
								<SelectItem key={m.key} value={m.key}>
									{m.displayName}
								</SelectItem>
							))}
						</SelectContent>
					</Select>

					<div className="flex justify-end">
						<Button
							disabled={confirmDisabled}
							onClick={handleConfirm}
							type="button"
						>
							{saving ? t("common.saving") : t("models.defaultModelButton")}
						</Button>
					</div>
				</div>
			</CardContent>
		</Card>
	);
}
