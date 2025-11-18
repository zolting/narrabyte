import type React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";

interface MarkdownRendererProps {
	content: string;
	className?: string;
}

export const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({
	content,
	className,
}) => {
	return (
		<div className={cn("text-xs leading-relaxed", className)}>
			<ReactMarkdown
				components={{
					h1: ({ children }) => (
						<h1 className="mt-3 mb-2 font-bold text-sm">{children}</h1>
					),
					h2: ({ children }) => (
						<h2 className="mt-3 mb-2 font-bold text-xs">{children}</h2>
					),
					h3: ({ children }) => (
						<h3 className="mt-2.5 mb-1.5 font-semibold text-xs">{children}</h3>
					),
					h4: ({ children }) => (
						<h4 className="mt-2 mb-1.5 font-semibold text-xs">{children}</h4>
					),
					h5: ({ children }) => (
						<h5 className="mt-2 mb-1 font-medium text-xs">{children}</h5>
					),
					h6: ({ children }) => (
						<h6 className="mt-1.5 mb-1 font-medium text-xs">{children}</h6>
					),
					p: ({ children }) => <p className="my-1.5">{children}</p>,
					ul: ({ children }) => (
						<ul className="my-1.5 list-disc pl-4">{children}</ul>
					),
					ol: ({ children }) => (
						<ol className="my-1.5 list-decimal pl-4 ml-3">{children}</ol>
					),
					li: ({ children }) => <li className="my-0.5">{children}</li>,
					code: ({ children, className: codeClassName }) => {
						const isInline = !codeClassName;
						return isInline ? (
							<code className="rounded bg-muted px-1 py-0.5 text-xs">
								{children}
							</code>
						) : (
							<code className={codeClassName}>{children}</code>
						);
					},
					pre: ({ children }) => (
						<pre className="my-2 overflow-x-auto rounded bg-muted p-2 text-xs">
							{children}
						</pre>
					),
					blockquote: ({ children }) => (
						<blockquote className="my-1.5 border-border border-l-2 pl-3 italic">
							{children}
						</blockquote>
					),
					hr: () => <hr className="my-2 border-border" />,
				}}
				remarkPlugins={[remarkGfm]}
			>
				{content}
			</ReactMarkdown>
		</div>
	);
};

export default MarkdownRenderer;
