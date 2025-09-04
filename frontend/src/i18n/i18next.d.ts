import "i18next";
import en from "../assets/locales/en.json" with { type: "json" };

declare module "i18next" {
	interface CustomTypeOptions {
		defaultNS: "translation";
		// Use the English translation shape as the source of truth for keys
		resources: {
			translation: typeof en;
		};
	}
}
