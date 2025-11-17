import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	createSessionKey,
	useDocGenerationStore,
} from "@/stores/docGeneration";

export type DocBranchConflictDialogProps = {
	projectId: number;
	projectName: string;
	sourceBranch: string;
	open: boolean;
	mode: "diff" | "single";
	targetBranch?: string;
	modelKey: string;
	userInstructions: string;
	existingDocsBranch: string;
	proposedDocsBranch: string;
	isInProgress?: boolean;
	sessionKey?: string;
};

export const DocBranchConflictDialog = ({
	projectId,
	projectName,
	sourceBranch,
	open,
	mode,
	targetBranch,
	modelKey,
	userInstructions,
	existingDocsBranch,
	proposedDocsBranch,
	isInProgress = false,
	sessionKey,
}: DocBranchConflictDialogProps) => {
	const { t } = useTranslation();

	const suggestName = useCallback(
		(name: string): string => {
			const base = (name ?? "").trim();
			const existing = (existingDocsBranch ?? "").trim();
			if (!base) {
				return base;
			}
			if (base === existing) {
				return base.endsWith("-2") ? base : `${base}-2`;
			}
			return base;
		},
		[existingDocsBranch]
	);

	const [newName, setNewName] = useState(suggestName(proposedDocsBranch));
	const [busy, setBusy] = useState(false);
	// Keep the local input state in sync when the dialog opens or the proposed branch changes
	useEffect(() => {
		if (open) {
			setNewName(suggestName(proposedDocsBranch));
		}
	}, [open, proposedDocsBranch, suggestName]);
  	const deleteAction = useDocGenerationStore(
  		(s) => s.resolveDocsBranchConflictByDelete
  	);
  	const renameAction = useDocGenerationStore(
  		(s) => s.resolveDocsBranchConflictByRename
  	);
  	const cancelAction = useDocGenerationStore((s) => s.cancel);
  	const clearConflict = useDocGenerationStore((s) => s.clearConflict);

	// Disable confirm when input is empty or same as the existing docs branch
	const sameAsExisting = useMemo(
		() => newName.trim() === existingDocsBranch.trim(),
		[existingDocsBranch, newName]
	);
	const disableConfirm = useMemo(
		() => busy || !newName.trim() || sameAsExisting,
		[busy, newName, sameAsExisting]
	);

	const handleClose = (next: boolean) => {
		if (!next) {
			const key = sessionKey ?? createSessionKey(projectId, sourceBranch ?? "");
			// If a generation session is in progress, ensure it is canceled
			if (isInProgress) {
				void cancelAction(projectId, key);
			}
			clearConflict(key);
		}
	};

	const handleDelete = async () => {
		setBusy(true);
		try {
			await deleteAction({
				projectId,
				projectName,
				sourceBranch,
				mode,
				targetBranch,
				modelKey,
				userInstructions,
				sessionKey,
			});
			handleClose(false);
		} finally {
			setBusy(false);
		}
	};

	const handleRename = async () => {
		const name = newName.trim();
		if (!name) {
			return;
		}
		setBusy(true);
		try {
			await renameAction({
				projectId,
				sourceBranch,
				newDocsBranch: name,
				mode,
				targetBranch,
				modelKey,
				userInstructions,
				sessionKey,
			});
			handleClose(false);
		} finally {
			setBusy(false);
		}
	};

	return (
		<Dialog onOpenChange={handleClose} open={open}>
			<DialogContent className="w-auto max-w-none sm:max-w-none md:max-w-none">
				<DialogHeader>
					<DialogTitle className="text-foreground text-lg">
						{t("common.docsBranchConflictTitle")}
					</DialogTitle>
					<DialogDescription className="text-muted-foreground">
						{isInProgress
							? t("common.docsBranchConflictInProgressDescription")
							: t("common.docsBranchConflictDescription")}
					</DialogDescription>
				</DialogHeader>
				<div className="flex flex-col gap-4">
					<div className="rounded-md border border-border bg-muted/30 p-3">
						<p className="text-foreground text-sm">
							{t("common.existingDocsBranch")}:{" "}
							<strong>{existingDocsBranch}</strong>
						</p>
					</div>
					<div className="flex flex-col gap-2">
						<Label htmlFor="newDocsBranch">
							{t("common.newDocsBranchLabel")}
						</Label>
						<Input
							aria-label={t("common.newDocsBranchAria")}
							id="newDocsBranch"
							onChange={(e) => setNewName(e.currentTarget.value)}
							placeholder={t("common.newDocsBranchPlaceholder")}
							value={newName}
						/>
					</div>
				</div>
				<DialogFooter className="gap-2 sm:flex-wrap sm:justify-between">
					<DialogClose asChild>
						<Button
							disabled={busy}
							onClick={() => handleClose(false)}
							type="button"
							variant="secondary"
						>
							{t("common.cancel")}
						</Button>
					</DialogClose>
					<div className="flex flex-wrap items-center gap-2">
						{!isInProgress && (
							<DialogClose asChild>
								<Button
									disabled={busy}
									onClick={handleDelete}
									type="button"
									variant="destructive"
								>
									{t("common.deleteCurrentDocsBranch")}
								</Button>
							</DialogClose>
						)}
						<DialogClose asChild>
							<Button
								disabled={disableConfirm}
								onClick={handleRename}
								type="button"
							>
								{t("common.useBranchOption", { branch: newName || "" })}
							</Button>
						</DialogClose>
					</div>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
};
