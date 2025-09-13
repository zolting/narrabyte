import { z } from "zod/v4";

// Zod schema for DemoEvent
export const demoEventSchema = z.object({
	id: z.number().int().positive(),
	type: z.enum(["info", "debug", "warn", "error"]),
	message: z.string().min(1),
	timestamp: z.coerce.date(),
});

export type DemoEvent = z.infer<typeof demoEventSchema>;
