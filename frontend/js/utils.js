// ========================================
// UTILITY FUNCTIONS
// ========================================

const API_BASE_URL = '/api/v1';
let userAccountNumber = null;
let currentFilter = 'all';
let todos = [];

// Get authentication headers with JWT token
function getAuthHeaders() {
    const headers = {
        'Content-Type': 'application/json'
    };
    
    // JWT token required for authentication (stored in sessionStorage, not localStorage)
    const token = sessionStorage.getItem('jwtToken');
    console.log('getAuthHeaders: token exists:', !!token, 'length:', token ? token.length : 0);
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
        console.log('getAuthHeaders: Authorization header set');
    } else {
        console.warn('getAuthHeaders: No JWT token found');
    }
    
    return headers;
}

// Copy text to clipboard
function copyToClipboard(text) {
    if (navigator.clipboard && window.isSecureContext) {
        return navigator.clipboard.writeText(text);
    } else {
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'absolute';
        textArea.style.left = '-999999px';
        document.body.prepend(textArea);
        textArea.select();
        try {
            document.execCommand('copy');
        } catch (error) {
            console.error('Failed to copy text:', error);
        } finally {
            textArea.remove();
        }
    }
}

// Debounce function for performance
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// ========================================
// ERROR MAPPING (SECURITY: Prevent information leakage)
// ========================================

// Map API error messages to generic user-facing messages
// SECURITY: Never show raw API error messages to users
const USER_ERROR_MESSAGES = {
    // Authentication errors
    'Invalid credentials': 'Invalid credentials',
    'Unauthorized': 'Unauthorized',
    'Invalid request': 'Invalid request',
    'Invalid or expired registration token': 'Unauthorized',
    'Account already created': 'Unauthorized',
    'Pending token required': 'Invalid request',
    
    // Generic errors
    'Internal server error': 'Internal server error',
    'Method not allowed': 'Method not allowed',
    'Not found': 'Invalid request',
    'Too many requests': 'Too many requests',
    'Forbidden': 'Forbidden',
    'Unsupported Media Type': 'Invalid request',
    'CORS origin not allowed': 'Forbidden',
    
    // Default fallback
    'default': 'Request failed'
};

// Get user-friendly error message from API response
// SECURITY: Always returns generic message, never leaks internal details
function getUserFriendlyError(apiMessage, statusCode) {
    if (!apiMessage) {
        // Map by status code if no message
        switch(statusCode) {
            case 400: return 'Invalid request';
            case 401: return 'Unauthorized';
            case 403: return 'Forbidden';
            case 404: return 'Invalid request';
            case 429: return 'Too many requests';
            case 500: return 'Internal server error';
            default: return USER_ERROR_MESSAGES.default;
        }
    }
    
    // Normalize message (case-insensitive, trim whitespace)
    const normalized = apiMessage.trim();
    
    // Check exact match first
    if (USER_ERROR_MESSAGES[normalized]) {
        return USER_ERROR_MESSAGES[normalized];
    }
    
    // Check case-insensitive match
    for (const [key, value] of Object.entries(USER_ERROR_MESSAGES)) {
        if (key.toLowerCase() === normalized.toLowerCase()) {
            return value;
        }
    }
    
    // Default fallback - never show raw API message
    return USER_ERROR_MESSAGES.default;
}

// ========================================
// ERROR HANDLING & STATUS PAGE REDIRECTS
// ========================================

// Handle error responses and redirect to appropriate status pages
async function handleErrorResponse(response, context = '') {
    const status = response.status;
    
    // Determine redirect URL first (before reading body)
    let redirectUrl = null;
    
    switch(status) {
        case 204:
            // No Content - usually successful DELETE, no redirect needed
            return; // Notification is sufficient
        
        case 400:
            redirectUrl = '/76e556a4-0575-4d98-a663-73b1080f26df.html';
            break;
        
        case 401:
            // Special handling: clear token, logout, then redirect
            sessionStorage.removeItem('jwtToken');
            sessionStorage.removeItem('userAccountNumber');
            redirectUrl = '/90b6b048-4bc9-4083-8585-9063c8e7332e.html';
            break;
        
        case 403:
            redirectUrl = '/cb012905-229f-4bf6-bdf4-8778ab34d8d7.html';
            break;
        
        case 404:
            redirectUrl = '/e0ab670f-6801-4475-b337-c08d5adb9e73.html';
            break;
        
        case 429:
            redirectUrl = '/8ec9c7e3-a58c-40b1-a918-f8cc67c4fc59.html';
            break;
        
        case 500:
            redirectUrl = '/9a24ed32-d32d-4b4b-a70e-a0bfd39f6710.html';
            break;
        
        default:
            // Unknown errors - show general error message
            // SECURITY: Never show status code to user
            if (typeof showError === 'function') {
                showError('Request failed');
            } else {
                console.error(`Unexpected error (${status}):`, context);
            }
            return; // No redirect for unknown errors
    }
    
    // If we have a redirect URL, perform redirect immediately
    // Use replace() instead of href to prevent back button issues
    if (redirectUrl) {
        // Read response body in background (non-blocking) to prevent potential issues
        // But don't wait for it - redirect immediately
        if (!response.bodyUsed) {
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                response.json().catch(() => {}); // Consume the body (fire and forget)
            } else {
                response.text().catch(() => {}); // Consume the body (fire and forget)
            }
        }
        
        // Perform redirect immediately using replace() to avoid history issues
        window.location.replace(redirectUrl);
    }
}

// Fetch wrapper with automatic error handling and status page redirects
async function fetchWithErrorHandling(url, options = {}) {
    try {
        const response = await fetch(url, options);
        
        if (!response.ok) {
            await handleErrorResponse(response, url);
            return null; // Return null on error to prevent further processing
        }
        
        return response;
    } catch (error) {
        // Network error - redirect to 500 error page immediately
        console.error('Network error:', error);
        window.location.replace('/9a24ed32-d32d-4b4b-a70e-a0bfd39f6710.html');
        return null;
    }
}