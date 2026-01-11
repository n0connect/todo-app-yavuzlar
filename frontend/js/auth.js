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
        errorElement.textContent = 'please enter uuid';
        return;
    }

    if (uuidInput.length !== 24) {
        errorElement.textContent = 'uuid must be 24 characters';
        return;
    }

    // Basic alphanumeric validation
    if (!/^[a-zA-Z0-9]{24}$/.test(uuidInput)) {
        errorElement.textContent = 'uuid must contain only letters and numbers';
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ uuid: uuidInput })
        });

        const data = await response.json();
        
        if (!data) {
            errorElement.textContent = 'invalid response';
            return;
        }

        if (response.ok && data.success) {
            userUUID = uuidInput;
            localStorage.setItem('userUUID', userUUID);
            
            if (data.token) {
                localStorage.setItem('jwtToken', data.token);
                console.log('JWT token stored successfully');
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
            errorElement.textContent = data.message || 'login failed';
        }
    } catch (error) {
        errorElement.textContent = 'login failed. please try again.';
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
    
    // Start continuous fake UUID generation
    fakeUUIDInterval = setInterval(() => {
        let fakeUUID = '';
        for (let i = 0; i < 24; i++) {
            // Mix of characters and occasional * for visual interest
            if (Math.random() < 0.08) {
                fakeUUID += '*';
            } else {
                fakeUUID += chars.charAt(Math.floor(Math.random() * chars.length));
            }
        }
        uuidElement.textContent = fakeUUID;
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
        const response = await fetch(`${API_BASE_URL}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ confirm: false })
        });

        const data = await response.json();
        
        if (!data || !data.success) {
            errorElement.textContent = data?.message || 'failed to generate uuid';
            uuidElement.textContent = 'error - try again';
            return;
        }

        const realUUID = data.uuid;
        const pendingToken = data.pending_token;
        
        if (!realUUID || realUUID.length !== 24) {
            errorElement.textContent = 'invalid uuid received';
            return;
        }
        
        if (!pendingToken) {
            errorElement.textContent = 'security token missing';
            return;
        }
        
        // Store pending UUID and token for account creation on continue
        pendingRegistrationUUID = realUUID;
        pendingRegistrationToken = pendingToken;
        
        // Reveal the real UUID with animation
        await revealRealUUID(uuidElement, realUUID);
        
        uuidElement.classList.remove('masking');
        uuidElement.classList.add('revealed');
        successElement.textContent = 'your uuid is ready! click continue to create account.';
        
        // Copy to clipboard
        try {
            await copyToClipboard(realUUID);
            showSuccess('UUID copied to clipboard!');
        } catch (e) {
            showWarning('Could not copy automatically. Please copy manually.');
        }
        
        // Enable continue button
        continueBtn.disabled = false;
        
    } catch (error) {
        errorElement.textContent = 'connection error. please try again.';
        console.error('UUID generation error:', error);
        uuidElement.textContent = 'error - try again';
    }
}

// Animate UUID to all asterisks
function animateToMask(uuidElement) {
    return new Promise((resolve) => {
        const currentText = uuidElement.textContent;
        let masked = currentText.split('');
        let step = 0;
        
        const maskInterval = setInterval(() => {
            // Mask 2-3 random characters per step
            for (let i = 0; i < 3; i++) {
                const randomIndex = Math.floor(Math.random() * 24);
                masked[randomIndex] = '*';
            }
            uuidElement.textContent = masked.join('');
            step++;
            
            // Check if all masked
            if (masked.every(c => c === '*') || step > 15) {
                clearInterval(maskInterval);
                uuidElement.textContent = '************************';
                resolve();
            }
        }, 50);
    });
}

// Reveal the real UUID character by character
function revealRealUUID(uuidElement, realUUID) {
    return new Promise((resolve) => {
        let revealed = '************************'.split('');
        let revealOrder = [];
        
        // Create random reveal order
        for (let i = 0; i < 24; i++) {
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
            for (let i = 0; i < 3 && step < 24; i++, step++) {
                const idx = revealOrder[step];
                revealed[idx] = realUUID[idx];
            }
            uuidElement.textContent = revealed.join('');
            
            if (step >= 24) {
                clearInterval(revealInterval);
                uuidElement.textContent = realUUID;
                resolve();
            }
        }, 40);
    });
}

async function continueWithUUID() {
    const generatedUUID = document.getElementById('generatedUUID').textContent;
    const successElement = document.getElementById('registerSuccess');
    const errorElement = document.getElementById('registerError');
    
    // Validate that we have a real UUID (not masked)
    if (!generatedUUID || generatedUUID.includes('*') || generatedUUID.length !== 24) {
        showError('Please click copy first to generate your UUID');
        return;
    }
    
    // Zero-trust: We need the pending token from Phase 1
    if (!pendingRegistrationToken) {
        showError('Security token expired. Please try again.');
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

        const data = await response.json();
        
        if (!response.ok || !data.success) {
            if (response.status === 401) {
                // Token expired or invalid
                errorElement.textContent = 'registration expired. please try again.';
                showError('Registration expired. Please start over.');
                setTimeout(() => cancelRegistration(), 2000);
            } else if (response.status === 409) {
                // UUID already taken (shouldn't happen with proper token)
                errorElement.textContent = 'account already exists.';
                showError('Account already created. Try logging in.');
            } else {
                errorElement.textContent = data?.message || 'account creation failed';
                showError('Failed to create account. Please try again.');
            }
            return;
        }
        
        // Account created successfully - use UUID from response (not frontend)
        userUUID = data.uuid;
        localStorage.setItem('userUUID', userUUID);
        
        // Clear pending state
        pendingRegistrationUUID = null;
        pendingRegistrationToken = null;
        
        // Store JWT token
        if (data.token) {
            localStorage.setItem('jwtToken', data.token);
            showSuccess('Welcome! Your account has been created.');
            showApp();
            loadTodos();
        } else {
            // Token not provided, try to login
            loginWithUUID(userUUID);
        }
        
    } catch (error) {
        console.error('Account creation error:', error);
        errorElement.textContent = 'connection error. please try again.';
        showError('Network error. Please try again.');
    }
}

// Helper function to login with UUID after registration
async function loginWithUUID(uuid) {
    try {
        const response = await fetch(`${API_BASE_URL}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ uuid: uuid })
        });

        const data = await response.json();
        
        if (response.ok && data.success && data.token) {
            localStorage.setItem('jwtToken', data.token);
            console.log('JWT token stored from post-registration login');
            showSuccess('Welcome! You\'re now logged in.');
            showApp();
            loadTodos();
        } else {
            showError('Failed to login automatically. Please login manually.');
            showMainMenu();
        }
    } catch (error) {
        console.error('Auto-login error:', error);
        showError('Network error during auto-login. Please login manually.');
        showMainMenu();
    }
}

function logout() {
    userUUID = null;
    localStorage.removeItem('userUUID');
    localStorage.removeItem('jwtToken');
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