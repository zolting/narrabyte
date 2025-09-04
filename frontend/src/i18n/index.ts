import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";

import en from "../assets/locales/en.json" with { type: "json" };
import fr from "../assets/locales/fr.json" with { type: "json" };

const resources = {
	en: { translation: en },
	fr: { translation: fr },
};

i18n
	.use(LanguageDetector)
	.use(initReactI18next)
	.init({
		resources,
		fallbackLng: {
			"en-CA": ["en"],
			"en-US": ["en"],
			"en-GB": ["en"],
			"fr-CA": ["fr"],
			"fr-FR": ["fr"],
			default: ["en"],
		},
		supportedLngs: ["en", "fr"],
		debug: false,
		interpolation: {
			escapeValue: false,
		},
	});
