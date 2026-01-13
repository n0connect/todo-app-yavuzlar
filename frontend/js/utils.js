// ========================================
// UTILITY FUNCTIONS
// ========================================

const API_BASE_URL = '/api/v1';
let userUUID = null;
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
// ERROR HANDLING & STATUS PAGE REDIRECTS
// ========================================

// Handle error responses and redirect to appropriate status pages
function handleErrorResponse(response, context = '') {
    const status = response.status;
    
    switch(status) {
        case 204:
            // No Content - usually successful DELETE, no redirect needed
            return; // Notification is sufficient
        
        case 400:
            window.location.href = '/76e556a4-0575-4d98-a663-73b1080f26df.html';
            break;
        
        case 401:
            // Special handling: clear token, logout, then redirect
            sessionStorage.removeItem('jwtToken');
            sessionStorage.removeItem('userUUID');
            window.location.href = '/90b6b048-4bc9-4083-8585-9063c8e7332e.html';
            break;
        
        case 403:
            window.location.href = '/cb012905-229f-4bf6-bdf4-8778ab34d8d7.html';
            break;
        
        case 404:
            window.location.href = '/e0ab670f-6801-4475-b337-c08d5adb9e73.html';
            break;
        
        case 429:
            window.location.href = '/8ec9c7e3-a58c-40b1-a918-f8cc67c4fc59.html';
            break;
        
        case 500:
            window.location.href = '/9a24ed32-d32d-4b4b-a70e-a0bfd39f6710.html';
            break;
        
        default:
            // Unknown errors - show general error message
            if (typeof showError === 'function') {
                showError(`Unexpected error (${status})`);
            } else {
                console.error(`Unexpected error (${status}):`, context);
            }
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
        // Network error - redirect to 500 error page
        console.error('Network error:', error);
        window.location.href = '/9a24ed32-d32d-4b4b-a70e-a0bfd39f6710.html';
        return null;
    }
}