import { AlertTriangle } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export type TabCloseConfirmDialogProps = {
	open: boolean;
	onConfirm: () => void;
	onCancel: () => void;
	tabIndex: number;
	sessionInfo?: {
		sourceBranch: string;
		isRunning: boolean;
	} | null;
};

export function TabCloseConfirmDialog({
	open,
	onConfirm,
	onCancel,
	tabIndex,
	sessionInfo,
}: TabCloseConfirmDialogProps) {
	const { t } = useTranslation();

	return (
		<AlertDialog onOpenChange={(isOpen) => !isOpen && onCancel()} open={open}>
			<AlertDialogContent>
				<AlertDialogHeader>
					<div className="flex items-center gap-3">
						<div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-destructive/10">
							<AlertTriangle className="h-5 w-5 text-destructive" />
						</div>
						<AlertDialogTitle className="text-foreground">
							{t("tabCloseConfirm.title", { index: tabIndex })}
						</AlertDialogTitle>
					</div>
					<AlertDialogDescription className="text-muted-foreground">
						{sessionInfo?.isRunning
							? t("tabCloseConfirm.descriptionRunning", {
									branch: sessionInfo.sourceBranch,
								})
							: t("tabCloseConfirm.description", {
									branch: sessionInfo?.sourceBranch || "",
								})}
					</AlertDialogDescription>
				</AlertDialogHeader>
				<AlertDialogFooter>
					<AlertDialogCancel onClick={onCancel} type="button">
						{t("common.cancel")}
					</AlertDialogCancel>
					<AlertDialogAction onClick={onConfirm} type="button">
						{t("tabCloseConfirm.closeTab")}
					</AlertDialogAction>
				</AlertDialogFooter>
			</AlertDialogContent>
		</AlertDialog>
	);
}
