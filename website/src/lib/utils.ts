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

/**
 * Capitalizes the first letter of each word in a string.
 */
export function capitalize(str: string): string {
    if (!str) return '';
    return str.split(' ')
              .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
              .join(' ');
}

/**
 * Formats a raw activity type string (e.g. 'PUBLISH_MATERIAL') into a human-readable title.
 */
export function formatActivityType(type: string): string {
    if (!type) return '';
    
    const mapping: Record<string, string> = {
        'BUILD_MATERIAL': 'Preparing Study Aid',
        'INGEST_DOCUMENTS': 'Analyzing Documents',
        'TRANSCRIBE_MEDIA': 'Processing Audio',
        'PUBLISH_MATERIAL': 'Exporting File',
        'SUGGEST': 'Refining Details',
        'DOWNLOAD_GOOGLE_DRIVE': 'Importing Files'
    };

    return mapping[type] || capitalize(type.replace(/_/g, ' '));
}
