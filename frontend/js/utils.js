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