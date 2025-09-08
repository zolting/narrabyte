import { useMemo, useState } from "react";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import { useTranslation } from "react-i18next";
import "react-diff-view/style/index.css";
import "./diff-view-theme.css";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from "@/components/ui/dialog";

// Placeholder git diff content
const SAMPLE_DIFF = `diff --git a/src/example.js b/src/example.js
index 1234567..abcdefg 100644
--- a/src/example.js
+++ b/src/example.js
@@ -1,8 +1,10 @@
 function greetUser(name) {
-    console.log("Hello, " + name);
+    console.log(\`Hello, \${name}!\`);
+    console.log("Welcome to the application");
 }
 
 function calculateTotal(items) {
-    return items.reduce((sum, item) => sum + item.price, 0);
+    const total = items.reduce((sum, item) => sum + item.price, 0);
+    return Math.round(total * 100) / 100; // Round to 2 decimal places
 }
 
+export { greetUser, calculateTotal };`;

interface GitDiffDialogProps {
	children: React.ReactNode;
}

function GitDiffDialog({ children }: GitDiffDialogProps) {
	const { t } = useTranslation();
	const [viewType, setViewType] = useState<"split" | "unified">("unified");

	// Memoize expensive diff parsing
	const files = useMemo(() => parseDiff(SAMPLE_DIFF), []);
	const file = useMemo(() => files[0], [files]);

	const toggleViewType = () => {
		setViewType((prevViewType) =>
			prevViewType === "split" ? "unified" : "split"
		);
	};

	return (
		<Dialog>
			<DialogTrigger asChild>{children}</DialogTrigger>
			<DialogContent className="flex max-h-[80vh] max-w-[95vw] flex-col overflow-hidden sm:max-w-[95vw] md:max-w-[90vw] lg:max-w-[85vw]">
				<DialogHeader>
					<DialogTitle className="text-foreground">
						{t("common.gitDiff")}
					</DialogTitle>
				</DialogHeader>
				<div className="mb-4 flex items-center justify-between">
					<div className="text-muted-foreground text-sm">
						{file?.newPath || "example.js"}
					</div>
					<Button
						className="border-border text-foreground hover:bg-accent hover:text-accent-foreground"
						onClick={toggleViewType}
						size="sm"
						variant="outline"
					>
						{viewType === "split"
							? t("common.inlineView")
							: t("common.splitView")}
					</Button>
				</div>
				<div className="diff-container flex-1 overflow-auto rounded-md border bg-background text-foreground">
					<Diff
						className="text-foreground"
						diffType={file?.type || "modify"}
						hunks={file?.hunks || []}
						lineClassName="text-foreground"
						optimizeSelection={false}
						viewType={viewType}
					>
						{(hunks) =>
							hunks.map((hunk) => <Hunk hunk={hunk} key={hunk.content} />)
						}
					</Diff>
				</div>
			</DialogContent>
		</Dialog>
	);
}

export { GitDiffDialog };
