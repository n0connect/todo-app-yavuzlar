# Not Simple Todo App - Encrypted Todo Application

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![JavaScript](https://img.shields.io/badge/JavaScript-ES6+-F7DF1E?style=flat&logo=javascript)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat&logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)
![Nginx](https://img.shields.io/badge/Nginx-Alpine-009639?style=flat&logo=nginx)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

Secure todo application with Bearer CSPRNG token-based AccountNumber-only authentication and per-user encryption.

<div align="center">
  <img src="c6acc3600c274500faddcc7f376d90a3faeb1f80cbe0dd119f29da9154462f54.png" alt="How we turned a simple todo app into this" style="max-width: 400px; width: 100%; height: auto;" />
</div>

---

## Overview

This application is a todo management system that authenticates using only cryptographic AccountNumber (no passwords) and encrypts all data with per-user encryption.

### Key Features

**Security:**
- AccountNumber-only authentication (192-bit entropy, Base64URL, 32 characters)
- AES-256-GCM encryption (title and tags encrypted)
- AAD (Additional Authenticated Data) for cross-user data swap protection
- Per-user encryption keys (unique key per user)
- Zero plaintext storage (AccountNumber never stored in plaintext in database)
- Rate limiting (IP-based, for login/register)
- Log masking (AccountNumber and sensitive information masked)
- Session storage (tokens in sessionStorage instead of localStorage)

**Functional:**
- Todo CRUD operations
- Tag system (encrypted)
- Due date
- Priority levels (low/medium/high)
- Filtering and sorting by tags

**Technical:**
- Frontend: Vanilla JavaScript (no framework dependencies)
- Backend: Go (net/http)
- Database: PostgreSQL
- Reverse Proxy: Nginx
- Containerization: Docker Compose

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ FRONTEND (Nginx + SPA)                                      │
│ - index.html, styles.css, js/{app,auth,todo,ui,utils}.js    │
│ - sessionStorage (JWT token, AccountNumber)                 │
│ - Progressive AccountNumber reveal                          │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP (Same-Origin Proxy)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ NGINX (Reverse Proxy)                                       │
│ - SPA routing (try_files)                                   │
│ - /api/* → backend:8080                                     │
│ - Security headers (CSP, X-Frame-Options, etc.)             │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ BACKEND (Go net/http)                                       │
│                                                             │
│ Middleware Chain:                                           │
│   SecurityHeaders → CORS → RateLimit → JWT Auth             │
│                                                             │
│ Handlers:                                                   │
│   Register (2-phase) │ Login │ GetTodos │ CreateTodo │ ...  │
│                                                             │
│ Service Layer:                                              │
│   TodoService: Validation → Encryption → Repository         │
│                                                             │
│ Core Modules:                                               │
│   Encryption (AES-256-GCM + AAD)                            │
│   Auth (JWT, Argon2id, HMAC lookup)                         │
│   Repository (GORM/PostgreSQL)                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ POSTGRESQL                                                  │
│                                                             │
│ users:                                                      │
│   - uuid (internal ID, 24 chars)                            │
│   - account_lookup (HMAC-SHA256(pepper, AccountNumber))     │
│   - account_hash (Argon2id(AccountNumber))                  │
│   - encrypted_key (AES-256-GCM encrypted user key)          │
│                                                             │
│ todos:                                                      │
│   - id, user_uuid, title(enc), tags(enc), due_date, priority│
└─────────────────────────────────────────────────────────────┘
```

---

## Pipeline and Workflow

### 1. Register Pipeline (2-Phase)

#### Phase 1: AccountNumber Generation

```
Frontend → POST /api/v1/register (body: {} or {"confirm": false})
    │
    ▼
Backend:
  1. GenerateAccountNumber()
     - crypto/rand: 24 bytes (192-bit entropy)
     - base64.RawURLEncoding → 32 chars Base64URL
     - Example: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
  
  2. ComputeAccountLookup(AccountNumber)
     - HMAC-SHA256(ACCOUNT_LOOKUP_PEPPER, AccountNumber)
     - Hex encoded → 64 chars
     - Collision check: ExistsByAccountLookup(lookup)
  
  3. GeneratePendingID()
     - crypto/rand: 16 bytes (128-bit)
     - Hex encoded → 32 chars
  
  4. StorePendingRegistration(pendingID, AccountNumber)
     - In-memory store (sync.RWMutex)
     - TTL: 5 minutes
     - One-time use (deleted after retrieval)
  
  5. SignPendingRegistrationToken(pendingID)
     - JWT token with pending_id (NOT AccountNumber)
     - Expiration: 5 minutes
     - Claims: { pending_id, is_pending: true }
    │
    ▼
Response:
  {
    "success": true,
    "message": "Account number generated - click continue to create account",
    "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq",
    "pending_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "confirmed": false
  }
    │
    ▼
Frontend:
  - Display AccountNumber to user (progressive reveal)
  - Store pending_token in memory
  - User clicks "Continue" → Phase 2
```

#### Phase 2: Account Creation

```
Frontend → POST /api/v1/register
  Body: {
    "confirm": true,
    "pending_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
    │
    ▼
Backend:
  1. VerifyPendingRegistrationToken(pendingToken)
     - Decode JWT → extract pending_id
     - GetPendingRegistration(pending_id)
       → Retrieve AccountNumber from server-side store
       → Delete entry (one-time use)
     - AccountNumber from token, NOT from user input
  
  2. ComputeAccountLookup(AccountNumber)
     - HMAC-SHA256(pepper, AccountNumber)
  
  3. ExistsByAccountLookup(lookup)
     - Check if account already exists (replay protection)
  
  4. HashAccountNumber(AccountNumber)
     - Argon2id hash
     - Parameters: memory=64MB, time=3, parallelism=2
     - Format: "$argon2id$v=19$m=67108864,t=3,p=2$salt$hash"
  
  5. GenerateSecureUUID()
     - Internal user ID (24 chars)
     - Used for JWT subject
  
  6. GenerateUserAESKey()
     - 32-byte AES-256 key
     - Encrypted with ENCRYPTION_KEY (master key)
     - Stored as EncryptedKey
  
  7. Create User:
     {
       UUID: internalUUID,
       AccountLookup: lookup,      // HMAC hash
       AccountHash: accountHash,    // Argon2id hash
       EncryptedKey: encryptedKey   // AES-256-GCM encrypted
     }
  
  8. SignToken(internalUUID)
     - JWT token (15 min expiration)
     - Subject: internal UUID (NOT AccountNumber)
    │
    ▼
Response:
  {
    "success": true,
    "message": "Account created successfully",
    "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "confirmed": true
  }
    │
    ▼
Frontend:
  - Store JWT token in sessionStorage
  - Store AccountNumber in sessionStorage
  - Redirect to app (loadTodos)
```

### 2. Login Pipeline

```
Frontend → POST /api/v1/login
  Body: {
    "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
  }
    │
    ▼
Backend:
  1. ValidateAccountNumber(AccountNumber)
     - Length: 32 chars
     - Regex: ^[A-Za-z0-9_-]{32}$ (Base64URL)
  
  2. ComputeAccountLookup(AccountNumber)
     - HMAC-SHA256(pepper, AccountNumber)
  
  3. FindByAccountLookup(lookup)
     - Query: SELECT * FROM users WHERE account_lookup = ?
  
  4. VerifyAccountNumberHash(user.AccountHash, AccountNumber)
     - Parse Argon2id hash format
     - Compute hash with same parameters
     - Constant-time comparison
  
  5. SignToken(user.UUID)
     - JWT token (15 min expiration)
     - Subject: internal UUID
    │
    ▼
Response:
  {
    "success": true,
    "message": "Login successful",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
  Note: account_number NOT in response (security)
    │
    ▼
Frontend:
  - Store JWT token in sessionStorage
  - Use existing AccountNumber from input
  - Redirect to app (loadTodos)
```

### 3. Todo CRUD Pipeline

#### Create Todo

```
Frontend → POST /api/v1/todos
  Headers: { Authorization: "Bearer <JWT>" }
  Body: {
    "title": "Buy groceries",
    "tags": ["shopping", "urgent"],
    "priority": "high",
    "due_date": "2026-01-15"
  }
    │
    ▼
Backend:
  1. JWTMiddleware
     - Extract JWT from Authorization header
     - Verify signature, expiration, issuer, audience
     - Extract user UUID from subject
     - Verify user exists in database
     - Set user context
  
  2. CreateTodoHandler
     - Validate request (title, priority, etc.)
     - Get user from context
  
  3. TodoService.CreateTodo()
     - Get user's encrypted key from DB
     - Decrypt user key with ENCRYPTION_KEY
     - Encrypt title with user key + AAD
     - Encrypt each tag with user key + AAD
     - Create todo record
  
  4. Repository.Create()
     - Insert into todos table
    │
    ▼
Response:
  {
    "id": "uuid",
    "title": "Buy groceries",  // Decrypted
    "tags": ["shopping", "urgent"],  // Decrypted
    "priority": "high",
    "due_date": "2026-01-15T00:00:00Z",
    "completed": false,
    "created_at": "2026-01-12T10:00:00Z"
  }
```

#### Get Todos

```
Frontend → GET /api/v1/todos
  Headers: { Authorization: "Bearer <JWT>" }
    │
    ▼
Backend:
  1. JWTMiddleware (same as above)
  
  2. GetTodosHandler
     - Get user from context
  
  3. TodoService.GetTodos()
     - Query: SELECT * FROM todos WHERE user_uuid = ?
     - Get user's encrypted key
     - Decrypt user key
     - For each todo:
       - Decrypt title with user key + AAD
       - Decrypt each tag with user key + AAD
    │
    ▼
Response:
  [
    {
      "id": "uuid1",
      "title": "Buy groceries",
      "tags": ["shopping", "urgent"],
      ...
    },
    ...
  ]
```

#### Update Todo

```
Frontend → PUT /api/v1/todos/{id}
  Headers: { Authorization: "Bearer <JWT>" }
  Body: {
    "title": "Buy groceries and milk",
    "completed": true,
    "priority": "medium",
    "tags": ["shopping"]
  }
    │
    ▼
Backend:
  1. JWTMiddleware
  
  2. UpdateTodoHandler
     - Parse todo ID from URL
     - Get user from context
  
  3. TodoService.UpdateTodo()
     - FindByIDAndUserUUID(todoID, userUUID) // Ownership check
     - Get user's encrypted key
     - Decrypt user key
     - Encrypt updated fields with user key + AAD
     - Update todo record
  
  4. Repository.Update()
     - WHERE id = ? AND user_uuid = ? // Ownership enforced
    │
    ▼
Response:
  {
    "id": "uuid",
    "title": "Buy groceries and milk",
    "completed": true,
    ...
  }
```

#### Delete Todo

```
Frontend → DELETE /api/v1/todos/{id}
  Headers: { Authorization: "Bearer <JWT>" }
    │
    ▼
Backend:
  1. JWTMiddleware
  
  2. DeleteTodoHandler
     - Parse todo ID from URL
     - Get user from context
  
  3. TodoService.DeleteTodo()
     - DeleteByIDAndUserUUID(todoID, userUUID) // Ownership check
    │
    ▼
Response:
  204 No Content
```

### 4. Encryption Pipeline

```
Layer 1: Master Key (Environment Variable)
  ENCRYPTION_KEY (32 bytes, hex encoded)
  ├── Used to encrypt/decrypt user-specific AES keys
  └── Never stored in database
    │
    ▼
Layer 2: User-Specific AES Key
  GenerateUserAESKey() → 32 bytes
  ├── Encrypted with ENCRYPTION_KEY (AES-256-GCM)
  └── Stored in users.encrypted_key
    │
    ▼
Layer 3: AAD (Additional Authenticated Data)
  BuildAAD(userUUID, todoID, fieldType)
  ├── Format: "PXAD" + version + userTag + todoID + field + purpose
  ├── 39 bytes total
  └── Prevents cross-user data swapping
    │
    ▼
Layer 4: Data Encryption
  EncryptWithAAD(userKey, plaintext, aad)
  ├── AES-256-GCM encryption
  ├── Nonce: 12 bytes (random per encryption)
  └── Output: nonce + ciphertext + tag (16 bytes)
    │
    ▼
Database Storage:
  todos.title: <nonce><ciphertext><tag> (binary, base64 encoded)
  todos.tags: JSON array of encrypted strings
```

---

## Security Model

### AccountNumber Generation

AccountNumber is generated with 192-bit cryptographic security:

1. `crypto/rand.Read(24 bytes)` → 192-bit entropy
2. `base64.RawURLEncoding.EncodeToString()` → 32 characters Base64URL
3. Result: 32 characters (A-Z, a-z, 0-9, -, _)
4. Example: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"

### AccountNumber Storage in Database

AccountNumber is NEVER stored in plaintext:

**HMAC Lookup (Fast Lookup):**
```
AccountNumber: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
    │
    ▼
HMAC-SHA256(ACCOUNT_LOOKUP_PEPPER, AccountNumber)
    │
    ▼
account_lookup: "b37e5a5c1f4e510a..." (64 hex chars)
  ├── Fast lookup (indexed)
  └── One-way function (cannot reverse)
```

**Argon2id Hash (Verification):**
```
AccountNumber: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
    │
    ▼
Argon2id(AccountNumber)
  ├── Memory: 64MB
  ├── Time: 3
  ├── Parallelism: 2
  └── Format: "$argon2id$v=19$m=67108864,t=3,p=2$salt$hash"
    │
    ▼
account_hash: "$argon2id$v=19$m=67108864,t=3,p=2$..." (stored)
  ├── Verification only
  └── Cannot reverse
```

### Rate Limiting

Token Bucket algorithm is used:
- Capacity: 20 requests
- Refill rate: 1 request/second
- Applied to: `/api/v1/login`, `/api/v1/register`
- IP-based tracking
- Response: `429 Too Many Requests`

### Log Masking

AccountNumber and sensitive information are masked in logs:
```
AccountNumber: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
    │
    ▼
MaskAccountNumber()
    │
    ▼
Log output: "****************************ATTDq"
  ├── Last 4 characters visible
  └── Rest masked with asterisks
```

### AAD Structure (39 bytes)

AAD (Additional Authenticated Data) is used to prevent cross-user data swap attacks:

```
┌────────┬─────┬──────────────┬──────────────┬───────┬─────────┐
│ MAGIC  │ VER │  USER_TAG    │   TODO_ID    │ FIELD │ PURPOSE │
│ "PXAD" │ 0x01│ SHA256[:16]  │  UUID (16B)  │ 1-3   │   1     │
│ 4 byte │ 1B  │   16 byte    │   16 byte    │  1B   │   1B    │
└────────┴─────┴──────────────┴──────────────┴───────┴─────────┘

FIELD: 0x01=Title, 0x02=Content, 0x03=Tags
PURPOSE: 0x01=Encryption
```

### Encrypted Fields

| Field | Encrypted | Description |
|-------|-----------|-------------|
| Title | Yes | AES-256-GCM + AAD |
| Tags | Yes | Each tag encrypted separately |
| Due Date | No | Metadata (date) |
| Priority | No | Metadata (enum) |
| Completed | No | Metadata (boolean) |
| AccountNumber | No | Plaintext not stored in DB; HMAC hash (account_lookup) and Argon2id hash (account_hash) stored |

### Content Security Policy

```
default-src 'self';
script-src 'self';                    # No inline scripts
style-src 'self' 'unsafe-inline';     # Inline styles for dynamic UI
font-src 'self';
connect-src 'self';
frame-ancestors 'none';               # No framing (clickjacking)
object-src 'none';                    # No plugins
```

---

## Installation

### Requirements

- Docker & Docker Compose
- Go 1.21+ (for development)
- OpenSSL (for setup.sh)

### Quick Start

```bash
# Clone repository
git clone <repo-url>
cd todo-app-yavuzlar

# Setup (automatically creates .env)
./setup.sh

# Build and test (all tests run, services won't start if tests fail)
./build.sh

# Access
open http://localhost
```

### Environment Variables

`setup.sh` automatically creates the following environment variables:

| Variable | Description | Production |
|----------|-------------|------------|
| `ENCRYPTION_KEY` | 32-byte hex encoded AES key (master key) | `openssl rand -hex 32` |
| `JWT_SECRET` | JWT signing key | `openssl rand -hex 32` |
| `ACCOUNT_LOOKUP_PEPPER` | Pepper for HMAC lookup (>=32 bytes) | `openssl rand -hex 32` |
| `JWT_EXPIRATION_MINUTES` | JWT token validity period | `15` (default) |
| `ALLOWED_ORIGIN` | CORS allowed origins | `*` (dev) or specific domain |
| `APP_ENV` | Environment mode | `development` or `production` |
| `LOG_LEVEL` | Log level | `debug`, `info`, `warn`, `error` |
| `DB_HOST` | PostgreSQL host | `postgres` (Docker) or `localhost` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `postgres` |
| `DB_NAME` | Database name | `todos` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_SSL_MODE` | SSL mode | `disable` (dev) or `require` (prod) |
| `BACKEND_PORT` | Backend server port | `8080` |

### Docker Compose Services

- `postgres`: PostgreSQL 15 (Alpine)
- `backend`: Go backend (Alpine)
- `frontend`: Nginx (Alpine) - SPA + API proxy

---

## API Reference

### Authentication

#### Register (Phase 1: Generate AccountNumber)

**Request:**
```
POST /api/v1/register
Content-Type: application/json

{}
```

**Response:**
```json
{
  "success": true,
  "message": "Account number generated - click continue to create account",
  "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq",
  "pending_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "confirmed": false
}
```

#### Register (Phase 2: Create Account)

**Request:**
```
POST /api/v1/register
Content-Type: application/json

{
  "confirm": true,
  "pending_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "success": true,
  "message": "Account created successfully",
  "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "confirmed": true
}
```

#### Login

**Request:**
```
POST /api/v1/login
Content-Type: application/json

{
  "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Login successful",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Note:** `account_number` is not in the response (security).

### Todos (JWT Required)

All todo endpoints require a JWT token. The token must be sent in the `Authorization` header as `Bearer <token>`.

#### Get Todos

**Request:**
```
GET /api/v1/todos
Authorization: Bearer <JWT>
```

**Response:**
```json
[
  {
    "id": "uuid",
    "title": "Buy groceries",
    "completed": false,
    "priority": "high",
    "due_date": "2026-01-15T00:00:00Z",
    "tags": ["shopping", "urgent"],
    "created_at": "2026-01-12T10:00:00Z"
  }
]
```

#### Create Todo

**Request:**
```
POST /api/v1/todos
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "title": "Buy groceries",
  "completed": false,
  "priority": "high",
  "due_date": "2026-01-15",
  "tags": ["shopping", "urgent"]
}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "Buy groceries",
  "completed": false,
  "priority": "high",
  "due_date": "2026-01-15T00:00:00Z",
  "tags": ["shopping", "urgent"],
  "created_at": "2026-01-12T10:00:00Z"
}
```

#### Update Todo

**Request:**
```
PUT /api/v1/todos/{id}
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "title": "Buy groceries and milk",
  "completed": true,
  "priority": "medium",
  "due_date": "2026-01-15",
  "tags": ["shopping"]
}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "Buy groceries and milk",
  "completed": true,
  "priority": "medium",
  "due_date": "2026-01-15T00:00:00Z",
  "tags": ["shopping"],
  "created_at": "2026-01-12T10:00:00Z"
}
```

#### Delete Todo

**Request:**
```
DELETE /api/v1/todos/{id}
Authorization: Bearer <JWT>
```

**Response:**
```
204 No Content
```

### Error Responses

**Format:**
```json
{
  "success": false,
  "message": "Error message"
}
```

**Status Codes:**
- `200 OK`: Success
- `201 Created`: Resource created
- `204 No Content`: Success (no body)
- `400 Bad Request`: Invalid request
- `401 Unauthorized`: Invalid credentials or expired token
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

---

## References

### Cryptography and Security

- **AES-256-GCM**: NIST SP 800-38D - Galois/Counter Mode (GCM) for Block Ciphers
- **Argon2id**: RFC 9106 - Argon2 Memory-Hard Function for Password Hashing and Key Derivation
- **HMAC-SHA256**: RFC 2104 - HMAC: Keyed-Hashing for Message Authentication
- **JWT**: RFC 7519 - JSON Web Token (JWT)
- **Base64URL**: RFC 4648 Section 5 - Base 64 Encoding with URL and Filename Safe Alphabet

### Zero-Trust

- **Zero-Trust Architecture**: NIST SP 800-207 - Zero Trust Architecture

### Technology Stack

- **Go**: https://go.dev/
- **PostgreSQL**: https://www.postgresql.org/
- **Nginx**: https://nginx.org/
- **Docker**: https://www.docker.com/
- **GORM**: https://gorm.io/

### Inspiration

- **Mullvad VPN**: AccountNumber-only authentication model
- **OWASP**: Security best practices
- **NIST**: Cryptographic standards

### Related Documentation

- **Go crypto/rand**: https://pkg.go.dev/crypto/rand
- **Go crypto/aes**: https://pkg.go.dev/crypto/aes
- **Go crypto/hmac**: https://pkg.go.dev/crypto/hmac
- **golang.org/x/crypto/argon2**: https://pkg.go.dev/golang.org/x/crypto/argon2
- **github.com/golang-jwt/jwt/v5**: https://pkg.go.dev/github.com/golang-jwt/jwt/v5

---

**Last Updated**: January 2026
**Version**: 2.0 (AccountNumber-based authentication)
