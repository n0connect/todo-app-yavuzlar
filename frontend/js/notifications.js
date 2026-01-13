// ========================================
// NOTIFICATION SYSTEM
// ========================================

function showNotification(message, type = 'info', duration = 4000) {
    const container = document.getElementById('notificationContainer');
    
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    
    const closeBtn = document.createElement('button');
    closeBtn.className = 'close-btn';
    closeBtn.textContent = '×';
    closeBtn.setAttribute('aria-label', 'Close notification');
    closeBtn.addEventListener('click', function() {
        closeNotification(this);
    });
    
    const msgDiv = document.createElement('div');
    msgDiv.textContent = message;
    
    notification.appendChild(closeBtn);
    notification.appendChild(msgDiv);
    
    // Add to container
    container.appendChild(notification);
    
    // Show animation
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);
    
    // Auto remove after duration
    if (duration > 0) {
        setTimeout(() => {
            closeNotification(notification.querySelector('.close-btn'));
        }, duration);
    }
}

function closeNotification(closeBtn) {
    const notification = closeBtn.parentElement;
    notification.classList.remove('show');
    
    // Remove from DOM after animation
    setTimeout(() => {
        if (notification.parentElement) {
            notification.parentElement.removeChild(notification);
        }
    }, 300);
}

function showSuccess(message) {
    showNotification(message, 'success');
}

function showError(message) {
    showNotification(message, 'error');
}

function showWarning(message) {
    showNotification(message, 'warning');
}