// ========================================
// FRONTEND INPUT VALIDATION (UX Layer)
// ========================================
// Security Strategy: OUTPUT ENCODING, not input filtering
// XSS Prevention: textContent auto-escapes HTML on render
// This allows all Unicode: emojis, <, >, quotes, special chars
// Backend stores as-is, encrypted with AES-256-GCM

const MAX_TITLE_LENGTH = 1000;
const MIN_TITLE_LENGTH = 1;

/**
 * Validates user input for todo titles
 * Minimal restrictions - allows all Unicode including emojis, <, >, etc.
 * XSS is prevented by output encoding (textContent), not input filtering
 * @param {string} input - User input
 * @returns {object} - {valid: boolean, sanitized: string, error: string}
 */
function validateTodoInput(input) {
    if (!input || typeof input !== 'string') {
        return { valid: false, sanitized: '', error: 'invalid input' };
    }

    const trimmed = input.trim();
    
    if (trimmed.length === 0) {
        return { valid: false, sanitized: '', error: 'todo cannot be empty' };
    }

    // Count Unicode characters (not bytes)
    const charCount = [...trimmed].length;
    
    if (charCount > MAX_TITLE_LENGTH) {
        return { valid: false, sanitized: '', error: `max ${MAX_TITLE_LENGTH} characters` };
    }

    // No character filtering - allow everything
    // XSS is prevented by textContent on render
    return { valid: true, sanitized: trimmed, error: '' };
}

/**
 * HTML escape function for display (defense-in-depth)
 * Backend already escapes, but this is extra safety
 * @param {string} str - String to escape
 * @returns {string} - Escaped string
 */
function escapeHtml(str) {
    if (!str || typeof str !== 'string') return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

/**
 * Safely set text content (prevents XSS)
 * Always use this instead of innerHTML for user content
 * @param {Element} element - DOM element
 * @param {string} text - Text to set
 */
function safeSetText(element, text) {
    if (element && typeof text === 'string') {
        element.textContent = text;
    }
}
