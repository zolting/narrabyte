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

	// Uncontrolled local selection fallback (when no defaultModelKey/onChange provided)
	const [localKey, setLocalKey] = useState<string>("");

	// Track last confirmed selection. Null = never confirmed => button active by default.
	const [confirmedKey, setConfirmedKey] = useState<string | null>(null);

	// Saving state for confirm action
	const [saving, setSaving] = useState(false);

	// Compute the current value for the Select:
	// 1) Use controlled prop if provided
	// 2) Else use local state
	// 3) Else first available model key (once loaded)
	const effectiveValue = useMemo(() => {
		if (defaultModelKey) return defaultModelKey;
		if (localKey) return localKey;
		return allModels[0]?.key ?? "";
	}, [defaultModelKey, localKey, allModels]);

	const handleSelect = (value: string) => {
		if (onChange) {
			onChange(value);
			return;
		}
		setLocalKey(value);
	};

	const handleConfirm = async () => {
		if (!effectiveValue || saving) return;
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
		saving || !effectiveValue || allModels.length === 0 || !isDirty;

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

				<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
					<Select
						disabled={loading || allModels.length === 0}
						value={effectiveValue}
						onValueChange={handleSelect}
					>
						<SelectTrigger className="w-full sm:w-[280px]">
							<SelectValue
								placeholder={"models.defaultModel.selectPlaceholder"}
							/>
						</SelectTrigger>
						<SelectContent>
							{allModels.map((m) => (
								<SelectItem key={m.key} value={m.key}>
									{m.displayName}
								</SelectItem>
							))}
						</SelectContent>
					</Select>

					<div className="flex justify-end">
						<Button
							type="button"
							onClick={handleConfirm}
							disabled={confirmDisabled}
						>
							{saving ? t("common.saving") : t("models.defaultModelButton")}
						</Button>
					</div>
				</div>
			</CardContent>
		</Card>
	);
}
