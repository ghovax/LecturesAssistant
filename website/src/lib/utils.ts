/**
 * Converts an ISO language code (e.g., 'en-US', 'ja-JP') into a human-readable language name.
 */
export function getLanguageName(code: string): string {
    try {
        const displayNames = new Intl.DisplayNames(['en'], { type: 'language' });
        return displayNames.of(code) || code;
    } catch (e) {
        return code;
    }
}
