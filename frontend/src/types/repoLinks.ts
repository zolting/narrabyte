import { z } from "zod/v4";

// Zod schema for RepoLink
export const repoLinkSchema = z.object({
	ID: z.number().int().positive(),
	ProjectName: z.string().min(1),
	DocumentationRepo: z.string().min(1),
	CodebaseRepo: z.string().min(1),
});

export type RepoLink = z.infer<typeof repoLinkSchema>;
