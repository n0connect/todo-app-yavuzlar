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

    // Pre-processing: pad message
    const ml = msg.length * 8;
    const msgPadded = new Uint8Array(msg);

    // Append the bit '1' to the message
    const paddedMsg = new Uint8Array(Math.ceil((msg.length + 9) / 64) * 64);
    paddedMsg.set(msgPadded);
    paddedMsg[msg.length] = 0x80; // Append '1' bit followed by zeros

    // Calculate the position for the length
    const pos = Math.floor((msg.length + 8) / 64) * 64;
    // Append original length in bits as 64-bit big-endian integer
    const view = new DataView(paddedMsg.buffer);
    view.setUint32(pos + 56, Math.floor(ml / 0x100000000), false); // High 32 bits
    view.setUint32(pos + 60, ml, false); // Low 32 bits

    // Process the message in 512-bit chunks
    const chunks = paddedMsg.buffer;
    const chunkSize = 64; // 512 bits
    const numChunks = chunks.byteLength / chunkSize;

    // Create a working copy of the hash values
    const HCopy = [...H];

    for (let chunkIndex = 0; chunkIndex < numChunks; chunkIndex++) {
        const chunkStart = chunkIndex * chunkSize;
        const chunkView = new DataView(chunks, chunkStart, chunkSize);

        // Create message schedule
        const W = new Array(64);
        for (let i = 0; i < 16; i++) {
            W[i] = chunkView.getUint32(i * 4, false); // Big endian
        }
        for (let i = 16; i < 64; i++) {
            const s0 = rotr(W[i - 15], 7) ^ rotr(W[i - 15], 18) ^ (W[i - 15] >>> 3);
            const s1 = rotr(W[i - 2], 17) ^ rotr(W[i - 2], 19) ^ (W[i - 2] >>> 10);
            W[i] = (W[i - 16] + s0 + W[i - 7] + s1) | 0;
        }

        // Initialize working variables
        let a = HCopy[0], b = HCopy[1], c = HCopy[2], d = HCopy[3];
        let e = HCopy[4], f = HCopy[5], g = HCopy[6], h = HCopy[7];

        // Main loop
        for (let i = 0; i < 64; i++) {
            const S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
            const ch = (e & f) ^ (~e & g);
            const temp1 = (h + S1 + ch + K[i] + W[i]) | 0;
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

        // Add compressed chunk to current hash value
        HCopy[0] = (HCopy[0] + a) | 0;
        HCopy[1] = (HCopy[1] + b) | 0;
        HCopy[2] = (HCopy[2] + c) | 0;
        HCopy[3] = (HCopy[3] + d) | 0;
        HCopy[4] = (HCopy[4] + e) | 0;
        HCopy[5] = (HCopy[5] + f) | 0;
        HCopy[6] = (HCopy[6] + g) | 0;
        HCopy[7] = (HCopy[7] + h) | 0;
    }

    // Produce the final hash value (big-endian)
    const result = new ArrayBuffer(32);
    const resultView = new DataView(result);
    for (let i = 0; i < 8; i++) {
        resultView.setUint32(i * 4, HCopy[i], false); // Big endian
    }

    // Convert to hex string
    const hex = Array.from(new Uint8Array(result))
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
    let byteIndex = 0;

    while (zeroBits < n && byteIndex * 2 < hexString.length) {
        const byteValue = parseInt(hexString.substr(byteIndex * 2, 2), 16);

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

        byteIndex++;
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
        error: 'Processing failed',
        attempts: maxAttempts
    });
};