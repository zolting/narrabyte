import { z } from "zod/v4";

// Zod schema for DemoEvent
export const demoEventSchema = z.object({
	id: z.string().uuid(),
	type: z.enum(["info", "debug", "warn", "error"]),
	message: z.string().min(1),
	timestamp: z.coerce.date(),
	sessionKey: z.string().optional(),
});

export type DemoEvent = z.infer<typeof demoEventSchema>;

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
