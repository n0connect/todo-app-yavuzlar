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

// ========================================
// WHITELIST VALIDATION FUNCTIONS
// ========================================

// Allowed priority values (whitelist)
const ALLOWED_PRIORITIES = ['low', 'medium', 'high'];

// Tag validation: alphanumeric, underscore, hyphen, max 50 chars
const MAX_TAG_LENGTH = 50;
const TAG_PATTERN = /^[a-zA-Z0-9_-]+$/;

/**
 * Validates priority value against whitelist
 * @param {string} priority - Priority value
 * @returns {object} - {valid: boolean, sanitized: string, error: string}
 */
function validatePriority(priority) {
    if (!priority || typeof priority !== 'string') {
        return { valid: false, sanitized: 'medium', error: 'invalid priority' };
    }
    
    const normalized = priority.toLowerCase().trim();
    
    if (!ALLOWED_PRIORITIES.includes(normalized)) {
        return { valid: false, sanitized: 'medium', error: 'invalid priority' };
    }
    
    return { valid: true, sanitized: normalized, error: '' };
}

/**
 * Validates a single tag
 * @param {string} tag - Tag string
 * @returns {object} - {valid: boolean, sanitized: string, error: string}
 */
function validateTag(tag) {
    if (!tag || typeof tag !== 'string') {
        return { valid: false, sanitized: '', error: 'invalid tag' };
    }
    
    const trimmed = tag.trim();
    
    if (trimmed.length === 0) {
        return { valid: false, sanitized: '', error: 'tag cannot be empty' };
    }
    
    if (trimmed.length > MAX_TAG_LENGTH) {
        return { valid: false, sanitized: '', error: `tag max ${MAX_TAG_LENGTH} characters` };
    }
    
    if (!TAG_PATTERN.test(trimmed)) {
        return { valid: false, sanitized: '', error: 'invalid tag format' };
    }
    
    return { valid: true, sanitized: trimmed, error: '' };
}

/**
 * Validates an array of tags
 * @param {Array<string>} tags - Array of tag strings
 * @returns {object} - {valid: boolean, sanitized: Array<string>, error: string}
 */
function validateTags(tags) {
    if (!Array.isArray(tags)) {
        return { valid: false, sanitized: [], error: 'tags must be an array' };
    }
    
    const sanitized = [];
    const seen = new Set();
    
    for (const tag of tags) {
        const validation = validateTag(tag);
        if (!validation.valid) {
            return { valid: false, sanitized: [], error: validation.error };
        }
        
        // Prevent duplicates (case-insensitive)
        const lowerTag = validation.sanitized.toLowerCase();
        if (seen.has(lowerTag)) {
            continue; // Skip duplicate
        }
        seen.add(lowerTag);
        sanitized.push(validation.sanitized);
    }
    
    // Limit total number of tags
    const MAX_TAGS = 10;
    if (sanitized.length > MAX_TAGS) {
        return { valid: false, sanitized: [], error: `max ${MAX_TAGS} tags allowed` };
    }
    
    return { valid: true, sanitized, error: '' };
}

/**
 * Validates date string (ISO 8601 date format: YYYY-MM-DD)
 * @param {string} dateStr - Date string
 * @returns {object} - {valid: boolean, sanitized: string|null, error: string}
 */
function validateDate(dateStr) {
    if (!dateStr || dateStr === '') {
        return { valid: true, sanitized: null, error: '' };
    }
    
    if (typeof dateStr !== 'string') {
        return { valid: false, sanitized: null, error: 'date must be a string' };
    }
    
    // ISO 8601 date format: YYYY-MM-DD
    const datePattern = /^\d{4}-\d{2}-\d{2}$/;
    if (!datePattern.test(dateStr)) {
        return { valid: false, sanitized: null, error: 'invalid date format' };
    }
    
    // Validate that it's a valid date
    const date = new Date(dateStr + 'T00:00:00');
    if (isNaN(date.getTime())) {
        return { valid: false, sanitized: null, error: 'invalid date' };
    }
    
    // Check if date string matches the parsed date (prevents invalid dates like 2023-13-45)
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const normalized = `${year}-${month}-${day}`;
    
    if (normalized !== dateStr) {
        return { valid: false, sanitized: null, error: 'invalid date' };
    }
    
    return { valid: true, sanitized: dateStr, error: '' };
}

/**
 * Validates todo ID (must be a valid number or string ID)
 * @param {any} id - Todo ID
 * @returns {boolean} - Whether ID is valid
 */
function validateTodoId(id) {
    if (id === null || id === undefined) {
        return false;
    }
    
    // Allow numeric IDs or non-empty string IDs
    if (typeof id === 'number' && !isNaN(id) && id > 0) {
        return true;
    }
    
    if (typeof id === 'string' && id.trim().length > 0) {
        return true;
    }
    
    return false;
}

// Allowed sort values (whitelist)
const ALLOWED_SORT_VALUES = ['created', 'due', 'priority', 'alpha'];

/**
 * Validates sort order value against whitelist
 * @param {string} sort - Sort value
 * @returns {object} - {valid: boolean, sanitized: string, error: string}
 */
function validateSortOrder(sort) {
    if (!sort || typeof sort !== 'string') {
        return { valid: false, sanitized: 'created', error: 'invalid sort value' };
    }
    
    const normalized = sort.toLowerCase().trim();
    
    if (!ALLOWED_SORT_VALUES.includes(normalized)) {
        return { valid: false, sanitized: 'created', error: 'invalid sort order' };
    }
    
    return { valid: true, sanitized: normalized, error: '' };
}

// Allowed filter values (whitelist)
const ALLOWED_FILTER_VALUES = ['all', 'active', 'completed'];

/**
 * Validates filter value against whitelist
 * @param {string} filter - Filter value
 * @returns {object} - {valid: boolean, sanitized: string, error: string}
 */
function validateFilter(filter) {
    if (!filter || typeof filter !== 'string') {
        return { valid: false, sanitized: 'all', error: 'invalid filter value' };
    }
    
    const normalized = filter.toLowerCase().trim();
    
    if (!ALLOWED_FILTER_VALUES.includes(normalized)) {
        return { valid: false, sanitized: 'all', error: 'invalid filter' };
    }
    
    return { valid: true, sanitized: normalized, error: '' };
}
