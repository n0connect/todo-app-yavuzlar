// ========================================
// AUTHENTICATION FUNCTIONS
// ========================================

async function login() {
    const accountNumberInputElement = document.getElementById('accountNumberInput');
    const errorElement = document.getElementById('loginError');
    const successElement = document.getElementById('loginSuccess');
    
    errorElement.textContent = '';
    successElement.textContent = '';

    // Check if eye animation is active and mask smoothly if needed
    if (window.eyeAnimation && window.eyeAnimation.currentZone !== 'none') {
        // Stop animation and get original value
        const originalValue = window.eyeAnimation.originalValue || accountNumberInputElement.value;
        window.eyeAnimation.stopAnimation();
        window.eyeAnimation.currentZone = 'none';
        window.eyeAnimation.resetToMasked();
        
        // Smoothly mask the input using existing animation
        await animateInputToMask(accountNumberInputElement, originalValue);
        
        // Restore original value after masking
        accountNumberInputElement.value = originalValue;
        accountNumberInputElement.type = 'password';
    }

    const accountNumberInput = accountNumberInputElement.value.trim();

    if (!accountNumberInput) {
        errorElement.textContent = 'please enter account number';
        return;
    }

    // Validate format without revealing exact rules
    if (accountNumberInput.length !== 32 || !/^[A-Za-z0-9_-]{32}$/.test(accountNumberInput)) {
        errorElement.textContent = 'invalid format';
        return;
    }

    try {
        const response = await fetchWithErrorHandling(`${API_BASE_URL}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ account_number: accountNumberInput })
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
            // Store account number
            userAccountNumber = accountNumberInput;
            sessionStorage.setItem('userAccountNumber', userAccountNumber);
            
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
let fakeAccountNumberInterval = null;
let pendingRegistrationAccountNumber = null;
let pendingRegistrationToken = null;  // Zero-trust: token from backend

// Fake Account Number generation animation - runs continuously until copy is clicked
function startFakeAccountNumberGeneration(accountNumberElement, successElement, accountNumberDisplay) {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    
    // Safety checks
    if (!accountNumberElement || !successElement || !accountNumberDisplay) {
        console.error('startFakeAccountNumberGeneration: missing required elements');
        return;
    }
    
    // Show the account number display area first
    accountNumberDisplay.style.display = 'block';
    const accountNumberBox = accountNumberDisplay.querySelector('.account-number-box');
    if (accountNumberBox) accountNumberBox.classList.add('show');
    
    successElement.textContent = 'Generating your special Account Number';
    accountNumberElement.classList.add('generating');
    
    // Clear any existing interval
    if (fakeAccountNumberInterval) {
        clearInterval(fakeAccountNumberInterval);
    }
    
    // Start continuous fake AccountNumber generation (base64url: A-Z, a-z, 0-9, _, -)
    const base64urlChars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-';
    fakeAccountNumberInterval = setInterval(() => {
        let fakeAccountNumber = '';
        for (let i = 0; i < 32; i++) {
            // Mix of base64url characters and occasional * for visual interest
            if (Math.random() < 0.08) {
                fakeAccountNumber += '*';
            } else {
                fakeAccountNumber += base64urlChars.charAt(Math.floor(Math.random() * base64urlChars.length));
            }
        }
        accountNumberElement.textContent = fakeAccountNumber;
    }, 120); // Fast scramble animation
}

// Stop the fake Account Number animation
function stopFakeAccountNumberAnimation() {
    if (fakeAccountNumberInterval) {
        clearInterval(fakeAccountNumberInterval);
        fakeAccountNumberInterval = null;
    }
}

// Copy button - get real Account Number from backend (but don't create account yet)
// Zero-trust: Backend returns a pending_token that we must use to confirm
async function copyAccountNumber() {
    const accountNumberElement = document.getElementById('generatedAccountNumber');
    const successElement = document.getElementById('registerSuccess');
    const errorElement = document.getElementById('registerError');
    const continueBtn = document.getElementById('continueBtn');
    const copyBtn = document.getElementById('copyBtn');
    
    // Disable copy button immediately to prevent multiple clicks
    if (copyBtn) {
        copyBtn.disabled = true;
    }
    
    // Stop the fake animation
    stopFakeAccountNumberAnimation();
    
    // Show masking animation
    accountNumberElement.classList.remove('generating');
    accountNumberElement.classList.add('masking');
    successElement.textContent = 'securing your identifier...';
    
    // Animate to all asterisks
    await animateToMask(accountNumberElement);
    
    // Get Account Number from backend (Phase 1 - no account creation)
    successElement.textContent = 'Generating your special Account Number';
    
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
            // Re-enable copy button on error
            if (copyBtn) {
                copyBtn.disabled = false;
            }
            return;
        }

        const data = await response.json();
        
        if (!data || !data.success) {
            errorElement.textContent = 'request failed';
            accountNumberElement.textContent = 'error - try again';
            // Re-enable copy button on error
            if (copyBtn) {
                copyBtn.disabled = false;
            }
            return;
        }

        const realAccountNumber = data.account_number;
        const pendingToken = data.pending_token;
        
        if (!realAccountNumber || realAccountNumber.length !== 32 || !pendingToken) {
            errorElement.textContent = 'request failed';
            // Re-enable copy button on error
            if (copyBtn) {
                copyBtn.disabled = false;
            }
            return;
        }
        
        // Store pending AccountNumber and token for account creation on continue
        pendingRegistrationAccountNumber = realAccountNumber;
        pendingRegistrationToken = pendingToken;
        
        // Reveal the real AccountNumber with animation
        await revealRealAccountNumber(accountNumberElement, realAccountNumber);
        
        accountNumberElement.classList.remove('masking');
        accountNumberElement.classList.add('revealed');
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
        accountNumberElement.textContent = 'error - try again';
        // Re-enable copy button on error
        if (copyBtn) {
            copyBtn.disabled = false;
        }
    }
}

// Animate AccountNumber to all asterisks (for display elements)
function animateToMask(accountNumberElement) {
    return new Promise((resolve) => {
        const currentText = accountNumberElement.textContent;
        let masked = currentText.split('');
        let step = 0;
        
        const maskInterval = setInterval(() => {
            // Mask 2-3 random characters per step
            for (let i = 0; i < 3; i++) {
                const randomIndex = Math.floor(Math.random() * 32);
                masked[randomIndex] = '*';
            }
            accountNumberElement.textContent = masked.join('');
            step++;
            
            // Check if all masked
            if (masked.every(c => c === '*') || step > 20) {
                clearInterval(maskInterval);
                accountNumberElement.textContent = '********************************';
                resolve();
            }
        }, 50);
    });
}

// Animate input field to all asterisks (for input elements)
function animateInputToMask(inputElement, originalValue) {
    return new Promise((resolve) => {
        const currentValue = inputElement.value || originalValue;
        let masked = currentValue.split('');
        let step = 0;
        
        // Show as text during animation
        inputElement.type = 'text';
        
        const maskInterval = setInterval(() => {
            // Mask 2-3 random characters per step
            for (let i = 0; i < 3; i++) {
                const randomIndex = Math.floor(Math.random() * masked.length);
                masked[randomIndex] = '*';
            }
            inputElement.value = masked.join('');
            step++;
            
            // Check if all masked
            if (masked.every(c => c === '*') || step > 20) {
                clearInterval(maskInterval);
                inputElement.value = '*'.repeat(masked.length);
                resolve();
            }
        }, 50);
    });
}

// Reveal the real AccountNumber character by character
function revealRealAccountNumber(accountNumberElement, realAccountNumber) {
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
            accountNumberElement.textContent = revealed.join('');
            
            if (step >= 32) {
                clearInterval(revealInterval);
                accountNumberElement.textContent = realAccountNumber;
                resolve();
            }
        }, 40);
    });
}

async function continueWithAccountNumber() {
    const generatedAccountNumber = document.getElementById('generatedAccountNumber').textContent;
    const successElement = document.getElementById('registerSuccess');
    const errorElement = document.getElementById('registerError');
    
    // Validate that we have a real AccountNumber (not masked)
    if (!generatedAccountNumber || generatedAccountNumber.includes('*') || generatedAccountNumber.length !== 32) {
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
        // Phase 2: Create account using pending token (not Account Number from frontend)
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
        userAccountNumber = data.account_number;
        sessionStorage.setItem('userAccountNumber', userAccountNumber);
        
        // Clear pending state
        pendingRegistrationAccountNumber = null;
        pendingRegistrationToken = null;
        
        // Store JWT token
        if (data.token) {
            sessionStorage.setItem('jwtToken', data.token);
            showSuccess('Welcome! Your account has been created.');
            showApp();
            loadTodos();
        } else {
            // Token not provided, try to login
            loginWithAccountNumber(userAccountNumber);
        }
        
    } catch (error) {
        console.error('Account creation error:', error);
        errorElement.textContent = 'request failed';
        showError('Please try again.');
    }
}

// Helper function to login with AccountNumber after registration
async function loginWithAccountNumber(accountNumber) {
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
    userAccountNumber = null;
    sessionStorage.removeItem('userAccountNumber');
    sessionStorage.removeItem('jwtToken');
    todos = [];
    showMainMenu();
    document.getElementById('accountNumberInput').value = '';
}

// Cancel registration - just go back (no account was created yet)
function cancelRegistration() {
    // Stop the fake animation
    stopFakeAccountNumberAnimation();
    
    // Clear any pending state
    pendingRegistrationAccountNumber = null;
    pendingRegistrationToken = null;
    
    // Return to main menu
    showMainMenu();
}