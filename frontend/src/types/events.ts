import { z } from "zod/v4";

// Zod schema for ToolEvent
export const toolEventSchema = z.object({
	id: z.string().uuid(),
	type: z.enum(["success", "info", "warn", "error"]),
	message: z.string().min(1),
	timestamp: z.coerce.date(),
	sessionKey: z.string().optional(),
});

export type ToolEvent = z.infer<typeof toolEventSchema>;

// Zod schema for TodoItem
export const todoItemSchema = z.object({
	content: z.string().min(1),
	activeForm: z.string().min(1),
	status: z.enum(["pending", "in_progress", "completed", "cancelled"]),
});

export type TodoItem = z.infer<typeof todoItemSchema>;

// Zod schema for TodoEvent
export const todoEventSchema = z.object({
	id: z.string().uuid(),
	todos: z.array(todoItemSchema),
	timestamp: z.coerce.date(),
	sessionKey: z.string().optional(),
});

export type TodoEvent = z.infer<typeof todoEventSchema>;
