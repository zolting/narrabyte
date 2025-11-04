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
				remarkPlugins={[remarkGfm]}
				components={{
				h1: ({ children }) => (
					<h1 className="text-sm font-bold mb-2 mt-3">{children}</h1>
				),
				h2: ({ children }) => (
					<h2 className="text-xs font-bold mb-2 mt-3">{children}</h2>
				),
				h3: ({ children }) => (
					<h3 className="text-xs font-semibold mb-1.5 mt-2.5">{children}</h3>
				),
				h4: ({ children }) => (
					<h4 className="text-xs font-semibold mb-1.5 mt-2">{children}</h4>
				),
				h5: ({ children }) => (
					<h5 className="text-xs font-medium mb-1 mt-2">{children}</h5>
				),
				h6: ({ children }) => (
					<h6 className="text-xs font-medium mb-1 mt-1.5">{children}</h6>
				),
				p: ({ children }) => <p className="my-1.5">{children}</p>,
				ul: ({ children }) => <ul className="my-1.5 list-disc pl-4">{children}</ul>,
				ol: ({ children }) => <ol className="my-1.5 list-decimal pl-4">{children}</ol>,
				li: ({ children }) => <li className="my-0.5">{children}</li>,
				code: ({ children, className: codeClassName }) => {
					const isInline = !codeClassName;
					return isInline ? (
						<code className="text-xs bg-muted px-1 py-0.5 rounded">
							{children}
						</code>
					) : (
						<code className={codeClassName}>{children}</code>
					);
				},
				pre: ({ children }) => (
					<pre className="text-xs bg-muted p-2 rounded my-2 overflow-x-auto">
						{children}
					</pre>
				),
				blockquote: ({ children }) => (
					<blockquote className="my-1.5 border-l-2 border-border pl-3 italic">
						{children}
					</blockquote>
				),
				hr: () => <hr className="my-2 border-border" />,
			}}
			>
				{content}
			</ReactMarkdown>
		</div>
	);
};

export default MarkdownRenderer;
