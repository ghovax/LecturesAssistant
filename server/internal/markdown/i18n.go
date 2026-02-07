package markdown

import (
	"fmt"
	"strings"
	"time"
)

var monthsMap = map[string][]string{
	"en": {"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	"it": {"Gennaio", "Febbraio", "Marzo", "Aprile", "Maggio", "Giugno", "Luglio", "Agosto", "Settembre", "Ottobre", "Novembre", "Dicembre"},
	"es": {"Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"},
	"fr": {"Janvier", "Février", "Mars", "Avril", "Mai", "Juin", "Juillet", "Août", "Septembre", "Octobre", "Novembre", "Décembre"},
	"de": {"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
	"pt": {"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho", "Julho", "Agosto", "Setembro", "Outubro", "Novembre", "Dezembro"},
}

func formatLocalizedDate(t time.Time, lang string) string {
	baseLang := strings.Split(lang, "-")[0]
	months, ok := monthsMap[baseLang]
	if !ok {
		months = monthsMap["en"]
	}

	month := months[t.Month()-1]

	switch baseLang {
	case "it", "es", "fr", "pt":
		// Day Month Year
		return fmt.Sprintf("%d %s %d", t.Day(), month, t.Year())
	case "de":
		// Day. Month Year
		return fmt.Sprintf("%d. %s %d", t.Day(), month, t.Year())
	default:
		// Default to English format: Month Day, Year
		return fmt.Sprintf("%s %d, %d", month, t.Day(), t.Year())
	}
}

var i18nMap = map[string]map[string]string{
	"en": {
		"abstract":        "abstract",
		"audio_files":     "Audio Files",
		"reference_files": "Reference Files",
		"page_label":      "p.",
		"pages_label":     "pp.",
	},
	"it": {
		"abstract":        "sommario",
		"audio_files":     "Registrazioni Audio",
		"reference_files": "Materiali di Riferimento",
		"page_label":      "p.",
		"pages_label":     "pp.",
	},
	"es": {
		"abstract":        "resumen",
		"audio_files":     "Archivos de Audio",
		"reference_files": "Materiales de Referencia",
		"page_label":      "pág.",
		"pages_label":     "págs.",
	},
	"fr": {
		"abstract":        "résumé",
		"audio_files":     "Fichiers Audio",
		"reference_files": "Documents de Référence",
		"page_label":      "p.",
		"pages_label":     "pp.",
	},
	"de": {
		"abstract":        "Zusammenfassung",
		"audio_files":     "Audiodateien",
		"reference_files": "Referenzmaterialien",
		"page_label":      "S.",
		"pages_label":     "S.",
	},
	"pt": {
		"abstract":        "resumo",
		"audio_files":     "Arquivos de Áudio",
		"reference_files": "Materiais de Referência",
		"page_label":      "p.",
		"pages_label":     "pp.",
	},
}

func getI18nLabel(lang, key string) string {
	if lang == "" {
		lang = "en"
	}
	// Extract base language (e.g., "en-US" -> "en")
	baseLang := strings.Split(lang, "-")[0]
	if labels, ok := i18nMap[baseLang]; ok {
		if label, ok := labels[key]; ok {
			return label
		}
	}
	// Fallback to English
	return i18nMap["en"][key]
}
