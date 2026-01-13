/* ========================================
   EYE HOVER PROGRESSIVE ACCOUNT NUMBER REVEAL SYSTEM
   5cm radius zone-based animation controller
   ======================================== */

class EyeHoverAnimation {
    constructor() {
        this.eyeIcon = null;
        this.accountNumberInput = null;
        this.isInitialized = false;
        this.currentZone = 'none';
        this.originalValue = '';
        this.displayValue = '';
        this.animationFrame = null;
        this.mouseSpeed = 0;
        this.lastMousePos = { x: 0, y: 0 };
        this.lastMouseTime = Date.now();
        this.revealProgress = 0; // 0 to 1
        
        // Zone configurations (in pixels)
        // Compact zones for precise control
        this.zones = {
            inner: 18,    // smooth reveal
            middle: 51,   // fast scramble (2x)
            outer: 87     // responsive scramble
        };
        
        this.characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        
        // Deterministic reveal order (seeded random)
        this.revealOrder = [];
        
        this.init();
    }

    init() {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.setupAnimation());
        } else {
            this.setupAnimation();
        }
    }

    setupAnimation() {
        this.eyeIcon = document.getElementById('eyeIcon');
        this.accountNumberInput = document.getElementById('accountNumberInput');
        
        if (!this.eyeIcon || !this.accountNumberInput) return;

        // Store original value when input changes
        this.accountNumberInput.addEventListener('input', (e) => {
            if (this.currentZone === 'none') {
                this.originalValue = e.target.value;
                this.generateRevealOrder();
            }
        });

        // Initialize
        this.originalValue = this.accountNumberInput.value;
        this.generateRevealOrder();

        // Track mouse on ENTIRE document (not just wrapper)
        document.addEventListener('mousemove', (e) => this.handleMouseMove(e));
        
        this.isInitialized = true;
    }

    generateRevealOrder() {
        // Create a shuffled order for character reveal
        const len = this.originalValue.length;
        this.revealOrder = [];
        for (let i = 0; i < len; i++) {
            this.revealOrder.push(i);
        }
        // Fisher-Yates shuffle with seed based on length
        for (let i = len - 1; i > 0; i--) {
            const j = (i * 7 + 3) % (i + 1);
            [this.revealOrder[i], this.revealOrder[j]] = [this.revealOrder[j], this.revealOrder[i]];
        }
    }

    handleMouseMove(e) {
        if (!this.eyeIcon || !this.accountNumberInput) return;
        
        // Only activate when login container is visible
        const loginContainer = document.getElementById('loginContainer');
        if (!loginContainer || loginContainer.style.display === 'none') return;
        
        // Calculate mouse speed
        const now = Date.now();
        const timeDelta = now - this.lastMouseTime;
        if (timeDelta > 0) {
            const dx = e.clientX - this.lastMousePos.x;
            const dy = e.clientY - this.lastMousePos.y;
            const distance = Math.sqrt(dx * dx + dy * dy);
            this.mouseSpeed = Math.min(distance / timeDelta * 10, 5); // Cap at 5
        }
        this.lastMousePos = { x: e.clientX, y: e.clientY };
        this.lastMouseTime = now;
        
        // Get eye icon center
        const rect = this.eyeIcon.getBoundingClientRect();
        const eyeCenterX = rect.left + rect.width / 2;
        const eyeCenterY = rect.top + rect.height / 2;
        
        // Calculate distance from mouse to eye center
        const distX = e.clientX - eyeCenterX;
        const distY = e.clientY - eyeCenterY;
        const distanceToEye = Math.sqrt(distX * distX + distY * distY);
        
        // Determine zone
        let newZone = 'none';
        if (distanceToEye <= this.zones.inner) {
            newZone = 'inner';
        } else if (distanceToEye <= this.zones.middle) {
            newZone = 'middle';
        } else if (distanceToEye <= this.zones.outer) {
            newZone = 'outer';
        }
        
        // Update zone
        if (newZone !== this.currentZone) {
            this.currentZone = newZone;
            this.updateEyeStyle();
            
            if (newZone === 'none') {
                this.stopAnimation();
                this.resetToMasked();
            } else {
                this.startAnimation();
            }
        }
    }

    updateEyeStyle() {
        this.eyeIcon.className = 'eye-icon';
        if (this.currentZone !== 'none') {
            this.eyeIcon.classList.add(`zone-${this.currentZone}`);
        }
    }

    startAnimation() {
        if (this.animationFrame) return;
        this.originalValue = this.accountNumberInput.value;
        if (!this.originalValue) return;
        this.generateRevealOrder();
        this.animate();
    }

    stopAnimation() {
        if (this.animationFrame) {
            cancelAnimationFrame(this.animationFrame);
            this.animationFrame = null;
        }
    }

    animate() {
        if (!this.originalValue || this.currentZone === 'none') {
            this.resetToMasked();
            return;
        }
        
        // Calculate animation parameters based on zone
        let scrambleIntensity = 0;
        let revealDelta = 0;
        
        switch (this.currentZone) {
            case 'outer':
                // Slow scramble, mouse speed responsive
                scrambleIntensity = 0.15 + this.mouseSpeed * 0.1;
                revealDelta = -0.02; // Move away from reveal
                break;
                
            case 'middle':
                // Fast scramble (2x), more intense
                scrambleIntensity = 0.4 + this.mouseSpeed * 0.15;
                revealDelta = 0.01; // Slight progress
                break;
                
            case 'inner':
                // Slow down and reveal smoothly
                scrambleIntensity = Math.max(0, 0.2 - this.revealProgress * 0.3);
                revealDelta = 0.03; // Move toward full reveal
                break;
        }
        
        // Update reveal progress
        this.revealProgress = Math.max(0, Math.min(1, this.revealProgress + revealDelta));
        
        // Generate display value
        this.displayValue = this.generateDisplayValue(scrambleIntensity);
        
        // Show as text and update display
        this.accountNumberInput.type = 'text';
        this.accountNumberInput.value = this.displayValue;
        
        // Continue animation
        this.animationFrame = requestAnimationFrame(() => this.animate());
    }

    generateDisplayValue(scrambleIntensity) {
        const len = this.originalValue.length;
        let result = new Array(len);
        
        // Calculate how many characters should be revealed based on progress
        const revealCount = Math.floor(this.revealProgress * len);
        
        for (let i = 0; i < len; i++) {
            // Check if this character position is in the revealed set
            const revealIndex = this.revealOrder.indexOf(i);
            const isRevealed = revealIndex < revealCount;
            
            if (isRevealed) {
                // Show real character
                result[i] = this.originalValue[i];
            } else if (Math.random() < scrambleIntensity) {
                // Scramble with random character
                result[i] = this.characters[Math.floor(Math.random() * this.characters.length)];
            } else {
                // Keep previous display value or show random
                result[i] = this.displayValue?.[i] || 
                           this.characters[Math.floor(Math.random() * this.characters.length)];
            }
        }
        
        return result.join('');
    }

    resetToMasked() {
        this.revealProgress = 0;
        this.accountNumberInput.type = 'password';
        this.accountNumberInput.value = this.originalValue;
        this.displayValue = '';
    }

    destroy() {
        this.stopAnimation();
        document.removeEventListener('mousemove', this.handleMouseMove);
    }
}

// Initialize
let eyeAnimation = null;

function initEyeAnimation() {
    const accountNumberInput = document.getElementById('accountNumberInput');
    const eyeIcon = document.getElementById('eyeIcon');
    
    if (accountNumberInput && eyeIcon && !eyeAnimation) {
        eyeAnimation = new EyeHoverAnimation();
        // Make eyeAnimation globally accessible for login function
        window.eyeAnimation = eyeAnimation;
    }
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initEyeAnimation);
} else {
    initEyeAnimation();
}

window.addEventListener('beforeunload', () => {
    if (eyeAnimation) eyeAnimation.destroy();
});
