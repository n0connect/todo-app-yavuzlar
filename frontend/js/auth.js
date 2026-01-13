// ========================================
// AUTHENTICATION FUNCTIONS
// ========================================

async function login() {
    const uuidInput = document.getElementById('uuidInput').value.trim();
    const errorElement = document.getElementById('loginError');
    const successElement = document.getElementById('loginSuccess');
    
    errorElement.textContent = '';
    successElement.textContent = '';

    if (!uuidInput) {
        errorElement.textContent = 'please enter account number';
        return;
    }

    // Validate format without revealing exact rules
    if (uuidInput.length !== 32 || !/^[A-Za-z0-9_-]{32}$/.test(uuidInput)) {
        errorElement.textContent = 'invalid format';
        return;
    }

    try {
        const response = await fetchWithErrorHandling(`${API_BASE_URL}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ account_number: uuidInput })
        });

        // If fetchWithErrorHandling returned null, it means error was handled and redirected
        if (!response) {
            return;
        }

        const data = await response.json();
        
        if (!data) {
            errorElement.textContent = 'request failed';
            return;
        }

        if (data.success) {
            // Store account number (for backward compatibility, still using userUUID variable name)
            userUUID = uuidInput; // Use input directly since response doesn't include account_number
            sessionStorage.setItem('userUUID', userUUID);
            
            if (data.token) {
                sessionStorage.setItem('jwtToken', data.token);
                console.log('JWT token stored successfully in sessionStorage');
            } else {
                console.warn('Login successful but no token received');
            }
            
            errorElement.textContent = '';
            successElement.textContent = 'login successful';
            setTimeout(() => {
                showApp();
                loadTodos();
            }, 500);
        } else {
            errorElement.textContent = 'login failed';
        }
    } catch (error) {
        errorElement.textContent = 'request failed';
        console.error('Login error:', error);
    }
}

// Global variables for registration state
let fakeUUIDInterval = null;
let pendingRegistrationUUID = null;
let pendingRegistrationToken = null;  // Zero-trust: token from backend

// Fake UUID generation animation - runs continuously until copy is clicked
function startFakeUUIDGeneration(uuidElement, successElement, uuidDisplay) {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    
    // Safety checks
    if (!uuidElement || !successElement || !uuidDisplay) {
        console.error('startFakeUUIDGeneration: missing required elements');
        return;
    }
    
    // Show the uuid display area first
    uuidDisplay.style.display = 'block';
    const uuidBox = uuidDisplay.querySelector('.uuid-box');
    if (uuidBox) uuidBox.classList.add('show');
    
    successElement.textContent = 'generating your unique identifier...';
    uuidElement.classList.add('generating');
    
    // Clear any existing interval
    if (fakeUUIDInterval) {
        clearInterval(fakeUUIDInterval);
    }
    
    // Start continuous fake AccountNumber generation (base64url: A-Z, a-z, 0-9, _, -)
    const base64urlChars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-';
    fakeUUIDInterval = setInterval(() => {
        let fakeAccountNumber = '';
        for (let i = 0; i < 32; i++) {
            // Mix of base64url characters and occasional * for visual interest
            if (Math.random() < 0.08) {
                fakeAccountNumber += '*';
            } else {
                fakeAccountNumber += base64urlChars.charAt(Math.floor(Math.random() * base64urlChars.length));
            }
        }
        uuidElement.textContent = fakeAccountNumber;
    }, 120); // Fast scramble animation
}

// Stop the fake UUID animation
function stopFakeUUIDAnimation() {
    if (fakeUUIDInterval) {
        clearInterval(fakeUUIDInterval);
        fakeUUIDInterval = null;
    }
}

// Copy button - get real UUID from backend (but don't create account yet)
// Zero-trust: Backend returns a pending_token that we must use to confirm
async function copyUUID() {
    const uuidElement = document.getElementById('generatedUUID');
    const successElement = document.getElementById('registerSuccess');
    const errorElement = document.getElementById('registerError');
    const continueBtn = document.getElementById('continueBtn');
    
    // Stop the fake animation
    stopFakeUUIDAnimation();
    
    // Show masking animation
    uuidElement.classList.remove('generating');
    uuidElement.classList.add('masking');
    successElement.textContent = 'securing your identifier...';
    
    // Animate to all asterisks
    await animateToMask(uuidElement);
    
    // Get UUID from backend (Phase 1 - no account creation)
    successElement.textContent = 'generating your unique identifier...';
    
    try {
        const response = await fetchWithErrorHandling(`${API_BASE_URL}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ confirm: false })
        });

        // If fetchWithErrorHandling returned null, it means error was handled and redirected
        if (!response) {
            return;
        }

        const data = await response.json();
        
        if (!data || !data.success) {
            errorElement.textContent = 'request failed';
            uuidElement.textContent = 'error - try again';
            return;
        }

        const realAccountNumber = data.account_number;
        const pendingToken = data.pending_token;
        
        if (!realAccountNumber || realAccountNumber.length !== 32 || !pendingToken) {
            errorElement.textContent = 'request failed';
            return;
        }
        
        // Store pending AccountNumber and token for account creation on continue
        pendingRegistrationUUID = realAccountNumber;
        pendingRegistrationToken = pendingToken;
        
        // Reveal the real AccountNumber with animation
        await revealRealUUID(uuidElement, realAccountNumber);
        
        uuidElement.classList.remove('masking');
        uuidElement.classList.add('revealed');
        successElement.textContent = 'your account number is ready! click continue to create account.';
        
        // Copy to clipboard
        try {
            await copyToClipboard(realAccountNumber);
            showSuccess('Account number copied to clipboard!');
        } catch (e) {
            showWarning('Could not copy automatically. Please copy manually.');
        }
        
        // Enable continue button
        continueBtn.disabled = false;
        
    } catch (error) {
        errorElement.textContent = 'request failed';
        console.error('AccountNumber generation error:', error);
        uuidElement.textContent = 'error - try again';
    }
}

// Animate AccountNumber to all asterisks
function animateToMask(uuidElement) {
    return new Promise((resolve) => {
        const currentText = uuidElement.textContent;
        let masked = currentText.split('');
        let step = 0;
        
        const maskInterval = setInterval(() => {
            // Mask 2-3 random characters per step
            for (let i = 0; i < 3; i++) {
                const randomIndex = Math.floor(Math.random() * 32);
                masked[randomIndex] = '*';
            }
            uuidElement.textContent = masked.join('');
            step++;
            
            // Check if all masked
            if (masked.every(c => c === '*') || step > 20) {
                clearInterval(maskInterval);
                uuidElement.textContent = '********************************';
                resolve();
            }
        }, 50);
    });
}

// Reveal the real AccountNumber character by character
function revealRealUUID(uuidElement, realAccountNumber) {
    return new Promise((resolve) => {
        let revealed = '********************************'.split('');
        let revealOrder = [];
        
        // Create random reveal order
        for (let i = 0; i < 32; i++) {
            revealOrder.push(i);
        }
        // Shuffle
        for (let i = revealOrder.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [revealOrder[i], revealOrder[j]] = [revealOrder[j], revealOrder[i]];
        }
        
        let step = 0;
        const revealInterval = setInterval(() => {
            // Reveal 2-3 characters per step
            for (let i = 0; i < 3 && step < 32; i++, step++) {
                const idx = revealOrder[step];
                revealed[idx] = realAccountNumber[idx];
            }
            uuidElement.textContent = revealed.join('');
            
            if (step >= 32) {
                clearInterval(revealInterval);
                uuidElement.textContent = realAccountNumber;
                resolve();
            }
        }, 40);
    });
}

async function continueWithUUID() {
    const generatedUUID = document.getElementById('generatedUUID').textContent;
    const successElement = document.getElementById('registerSuccess');
    const errorElement = document.getElementById('registerError');
    
    // Validate that we have a real AccountNumber (not masked)
    if (!generatedUUID || generatedUUID.includes('*') || generatedUUID.length !== 32) {
        showError('Please generate your account number first');
        return;
    }
    
    // Zero-trust: We need the pending token from Phase 1
    if (!pendingRegistrationToken) {
        showError('Session expired. Please try again.');
        cancelRegistration();
        return;
    }
    
    successElement.textContent = 'creating your account...';
    errorElement.textContent = '';
    
    try {
        // Phase 2: Create account using pending token (not UUID from frontend)
        const response = await fetch(`${API_BASE_URL}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ 
                confirm: true, 
                pending_token: pendingRegistrationToken 
            })
        });

        if (!response.ok) {
            // Handle 409 Conflict specially (no status page for this)
            if (response.status === 409) {
                const data = await response.json().catch(() => ({}));
                errorElement.textContent = 'operation failed';
                showError('Please try logging in instead.');
                return;
            }
            // For other errors, use error handler (will redirect to status pages)
            await handleErrorResponse(response, 'register');
            return;
        }

        const data = await response.json();
        
        if (!data.success) {
            errorElement.textContent = 'operation failed';
            showError('Please try again.');
            return;
        }
        
        // Account created successfully - use AccountNumber from response (not frontend)
        userUUID = data.account_number;
        sessionStorage.setItem('userUUID', userUUID);
        
        // Clear pending state
        pendingRegistrationUUID = null;
        pendingRegistrationToken = null;
        
        // Store JWT token
        if (data.token) {
            sessionStorage.setItem('jwtToken', data.token);
            showSuccess('Welcome! Your account has been created.');
            showApp();
            loadTodos();
        } else {
            // Token not provided, try to login
            loginWithUUID(userUUID);
        }
        
    } catch (error) {
        console.error('Account creation error:', error);
        errorElement.textContent = 'request failed';
        showError('Please try again.');
    }
}

// Helper function to login with AccountNumber after registration
async function loginWithUUID(accountNumber) {
    try {
        const response = await fetchWithErrorHandling(`${API_BASE_URL}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ account_number: accountNumber })
        });

        // If fetchWithErrorHandling returned null, it means error was handled and redirected
        if (!response) {
            return;
        }

        const data = await response.json();
        
        if (data.success && data.token) {
            sessionStorage.setItem('jwtToken', data.token);
            console.log('JWT token stored from post-registration login');
            showSuccess('Welcome! You\'re now logged in.');
            showApp();
            loadTodos();
        } else {
            showError('Please login manually.');
            showMainMenu();
        }
    } catch (error) {
        console.error('Auto-login error:', error);
        showError('Please login manually.');
        showMainMenu();
    }
}

function logout() {
    userUUID = null;
    sessionStorage.removeItem('userUUID');
    sessionStorage.removeItem('jwtToken');
    todos = [];
    showMainMenu();
    document.getElementById('uuidInput').value = '';
}

// Cancel registration - just go back (no account was created yet)
function cancelRegistration() {
    // Stop the fake animation
    stopFakeUUIDAnimation();
    
    // Clear any pending state
    pendingRegistrationUUID = null;
    pendingRegistrationToken = null;
    
    // Return to main menu
    showMainMenu();
}