/* ========================================
   SECURE TEXT HOVER SCRAMBLE ANIMATION
   Zone-based reveal triggered by login button proximity
   ======================================== */

class SecureTextAnimation {
    constructor() {
        this.elements = [];
        this.characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%&*';
        this.activeElement = null;
        this.triggerRadius = 120; // pixels - covers login button area
        this.init();
    }

    init() {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.setup());
        } else {
            this.setup();
        }
    }

    setup() {
        this.elements = document.querySelectorAll('.secure-text');
        
        this.elements.forEach(el => {
            const originalText = el.dataset.text || el.textContent;
            el.dataset.text = originalText;
            el.isAnimating = false;
        });

        // Track mouse on entire document
        document.addEventListener('mousemove', (e) => this.handleMouseMove(e));
    }

    handleMouseMove(e) {
        this.elements.forEach(el => {
            // Check if parent container is visible
            const container = el.closest('.main-menu, .login-container, .register-container');
            if (!container || container.style.display === 'none') {
                if (el.isAnimating) {
                    this.stopScramble(el);
                }
                return;
            }

            // Get element center
            const rect = el.getBoundingClientRect();
            const centerX = rect.left + rect.width / 2;
            const centerY = rect.top + rect.height / 2;

            // Calculate distance
            const distX = e.clientX - centerX;
            const distY = e.clientY - centerY;
            const distance = Math.sqrt(distX * distX + distY * distY);

            // Check if within trigger radius
            if (distance <= this.triggerRadius) {
                if (!el.isAnimating) {
                    this.startScramble(el);
                }
            } else {
                if (el.isAnimating) {
                    this.stopScramble(el);
                }
            }
        });
    }

    startScramble(element) {
        if (element.isAnimating) return;
        
        const originalText = element.dataset.text;
        const length = originalText.length;
        let iteration = 0;
        const maxIterations = 15;
        
        if (element.scrambleInterval) {
            clearInterval(element.scrambleInterval);
        }
        
        element.isAnimating = true;
        element.classList.add('scrambling');
        
        element.scrambleInterval = setInterval(() => {
            const revealCount = Math.floor((iteration / maxIterations) * length);
            
            let display = '';
            for (let i = 0; i < length; i++) {
                if (i < revealCount) {
                    display += originalText[i];
                } else {
                    display += this.characters[Math.floor(Math.random() * this.characters.length)];
                }
            }
            
            element.textContent = display;
            iteration++;
            
            if (iteration > maxIterations) {
                element.textContent = originalText;
                element.classList.remove('scrambling');
                element.classList.add('revealed');
                clearInterval(element.scrambleInterval);
                element.scrambleInterval = null;
                // Keep animating state true to prevent re-trigger while in zone
            }
        }, 50);
    }

    stopScramble(element) {
        const originalText = element.dataset.text;
        
        if (element.scrambleInterval) {
            clearInterval(element.scrambleInterval);
            element.scrambleInterval = null;
        }
        
        element.isAnimating = false;
        
        // Quick reverse scramble
        let iteration = 0;
        const frames = 8;
        
        element.classList.remove('revealed');
        
        const reverseInterval = setInterval(() => {
            if (iteration >= frames) {
                element.textContent = originalText;
                element.classList.remove('scrambling');
                clearInterval(reverseInterval);
                return;
            }
            
            let display = '';
            for (let i = 0; i < originalText.length; i++) {
                if (Math.random() < 0.4) {
                    display += this.characters[Math.floor(Math.random() * this.characters.length)];
                } else {
                    display += originalText[i];
                }
            }
            element.textContent = display;
            iteration++;
        }, 40);
    }
}

// Initialize
new SecureTextAnimation();
