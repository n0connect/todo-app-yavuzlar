// pow-worker.js
// Web Worker for Proof of Work computation

// Pure JavaScript SHA-256 implementation
function sha256(message) {
    // Convert string to byte array
    const msg = new TextEncoder().encode(message);
    
    // Constants
    const K = [
        0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
        0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
        0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
        0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
        0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
        0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
        0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
        0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
    ];
    
    // Initial hash values
    const H = [
        0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
        0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19
    ];
    
    // Pre-processing
    const ml = msg.length * 8;
    msg.push(0x80);
    
    while ((msg.length * 8) % 512 !== 448) {
        msg.push(0x00);
    }
    
    // Append length as 64-bit big-endian
    for (let i = 0; i < 8; i++) {
        msg.push((ml >>> (56 - i * 8)) & 0xff);
    }
    
    // Process the message in successive 512-bit chunks
    for (let i = 0; i < msg.length; i += 64) {
        const W = new Array(64);
        
        // Break chunk into sixteen 32-bit big-endian words
        for (let j = 0; j < 16; j++) {
            W[j] = (msg[i + j * 4] << 24) |
                   (msg[i + j * 4 + 1] << 16) |
                   (msg[i + j * 4 + 2] << 8) |
                   (msg[i + j * 4 + 3]);
        }
        
        // Extend the sixteen 32-bit words into sixty-four 32-bit words
        for (let j = 16; j < 64; j++) {
            const s0 = rotr(W[j - 15], 7) ^ rotr(W[j - 15], 18) ^ (W[j - 15] >>> 3);
            const s1 = rotr(W[j - 2], 17) ^ rotr(W[j - 2], 19) ^ (W[j - 2] >>> 10);
            W[j] = (W[j - 16] + s0 + W[j - 7] + s1) | 0;
        }
        
        // Initialize working variables
        let a = H[0], b = H[1], c = H[2], d = H[3];
        let e = H[4], f = H[5], g = H[6], h = H[7];
        
        // Main loop
        for (let j = 0; j < 64; j++) {
            const S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
            const ch = (e & f) ^ (~e & g);
            const temp1 = (h + S1 + ch + K[j] + W[j]) | 0;
            const S0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
            const maj = (a & b) ^ (a & c) ^ (b & c);
            const temp2 = (S0 + maj) | 0;
            
            h = g;
            g = f;
            f = e;
            e = (d + temp1) | 0;
            d = c;
            c = b;
            b = a;
            a = (temp1 + temp2) | 0;
        }
        
        // Add this chunk's hash to result so far
        H[0] = (H[0] + a) | 0;
        H[1] = (H[1] + b) | 0;
        H[2] = (H[2] + c) | 0;
        H[3] = (H[3] + d) | 0;
        H[4] = (H[4] + e) | 0;
        H[5] = (H[5] + f) | 0;
        H[6] = (H[6] + g) | 0;
        H[7] = (H[7] + h) | 0;
    }
    
    // Produce the final hash value (big-endian)
    const hash = new ArrayBuffer(32);
    const view = new DataView(hash);
    for (let i = 0; i < 8; i++) {
        view.setUint32(i * 4, H[i]);
    }
    
    // Convert to hex string
    const hex = Array.from(new Uint8Array(hash))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
    
    return hex;
}

function rotr(x, n) {
    return ((x >>> n) | (x << (32 - n))) | 0;
}

// Check if a hex string has at least n leading zero bits
function hasNLeadingZeros(hexString, n) {
    if (n <= 0) return true;
    
    let zeroBits = 0;
    let hexIndex = 0;
    
    while (zeroBits < n && hexIndex < hexString.length) {
        const byteValue = parseInt(hexString.substr(hexIndex * 2, 2), 16);
        
        if (byteValue === 0) {
            zeroBits += 8;
        } else {
            // Count leading zeros in this byte
            for (let bit = 7; bit >= 0; bit--) {
                if ((byteValue >> bit) & 1) {
                    break;
                }
                zeroBits++;
            }
            break;
        }
        
        hexIndex++;
    }
    
    return zeroBits >= n;
}

// Main message handler for the Web Worker
self.onmessage = function(e) {
    const { challenge, difficulty, salt } = e.data;
    
    // Maximum attempts to prevent infinite loops
    const maxAttempts = 10000000;
    
    for (let nonce = 0; nonce < maxAttempts; nonce++) {
        // Create the input for hashing: challenge + nonce + salt
        const input = challenge + nonce.toString() + salt;
        
        const hash = sha256(input);
        
        if (hasNLeadingZeros(hash, difficulty)) {
            // Found a valid solution
            self.postMessage({ 
                success: true, 
                solution: nonce.toString(),
                attempts: nonce
            });
            return;
        }
        
        // Occasionally yield control back to the browser
        if (nonce % 10000 === 0) {
            // Post progress update
            self.postMessage({ 
                progress: true,
                attempts: nonce,
                maxAttempts: maxAttempts
            });
        }
    }
    
    // If we reach here, no solution was found within max attempts
    self.postMessage({ 
        success: false, 
        error: 'No solution found within max attempts',
        attempts: maxAttempts
    });
};