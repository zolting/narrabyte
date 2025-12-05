import { Loader2, Trash2 } from "lucide-react";
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
	AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

interface DeleteSessionDialogProps {
	onConfirm: () => void;
	branchName?: string | null;
	isDeleting?: boolean;
	disabled?: boolean;
	// Controlled mode props (optional)
	open?: boolean;
	onOpenChange?: (open: boolean) => void;
}

export function DeleteSessionDialog({
	onConfirm,
	branchName,
	isDeleting = false,
	disabled = false,
	open,
	onOpenChange,
}: DeleteSessionDialogProps) {
	const { t } = useTranslation();

	const content = (
		<AlertDialogContent>
			<AlertDialogHeader>
				<AlertDialogTitle>{t("generations.deleteSession")}</AlertDialogTitle>
				<AlertDialogDescription className="whitespace-pre-line">
					{branchName
						? t("common.deleteSessionConfirmDescription", {
								branch: branchName,
							})
						: t("generations.deleteConfirm")}
				</AlertDialogDescription>
			</AlertDialogHeader>
			<AlertDialogFooter>
				<AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
				<AlertDialogAction
					className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
					onClick={onConfirm}
				>
					{t("common.delete")}
				</AlertDialogAction>
			</AlertDialogFooter>
		</AlertDialogContent>
	);

	// Controlled mode (with open/onOpenChange props)
	if (open !== undefined && onOpenChange) {
		return (
			<AlertDialog onOpenChange={onOpenChange} open={open}>
				{content}
			</AlertDialog>
		);
	}

	// Trigger mode (default)
	return (
		<AlertDialog>
			<AlertDialogTrigger asChild>
				<Button
					disabled={disabled || isDeleting}
					size="sm"
					type="button"
					variant="destructive"
				>
					{isDeleting ? (
						<Loader2 className="mr-2 h-4 w-4 animate-spin" />
					) : (
						<Trash2 className="mr-2 h-4 w-4" />
					)}
					{t("common.delete")}
				</Button>
			</AlertDialogTrigger>
			{content}
		</AlertDialog>
	);
}
