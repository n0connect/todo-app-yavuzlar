# Not Simple Todo App - Encrypted Todo Application

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![JavaScript](https://img.shields.io/badge/JavaScript-ES6+-F7DF1E?style=flat&logo=javascript)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat&logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)
![Nginx](https://img.shields.io/badge/Nginx-Alpine-009639?style=flat&logo=nginx)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

Bearer CSPRNG token tabanlı AccountNumber-only kimlik doğrulama ve per-user encryption ile güvenli todo uygulaması.

---

## Genel Bakış

Bu uygulama, şifre kullanmadan sadece kriptografik AccountNumber ile kimlik doğrulama yapan, tüm verileri per-user encryption ile şifreleyen bir todo yönetim sistemidir.

### Temel Özellikler

**Güvenlik:**
- AccountNumber-only authentication (192-bit entropy, Base64URL, 32 karakter)
- AES-256-GCM encryption (title ve tags şifreli)
- AAD (Additional Authenticated Data) ile cross-user data swap koruması
- Per-user encryption keys (her kullanıcıya özel anahtar)
- Zero plaintext storage (AccountNumber veritabanında plaintext tutulmaz)
- Rate limiting (IP bazlı, login/register için)
- Log masking (AccountNumber ve hassas bilgiler maskelenir)
- Session storage (token'lar localStorage yerine sessionStorage'da)

**Fonksiyonel:**
- Todo CRUD işlemleri
- Tag sistemi (şifreli)
- Due date (son tarih)
- Priority seviyeleri (low/medium/high)
- Tag ile filtreleme ve sıralama

**Teknik:**
- Frontend: Vanilla JavaScript (framework bağımlılığı yok)
- Backend: Go (net/http)
- Database: PostgreSQL
- Reverse Proxy: Nginx
- Containerization: Docker Compose

---

## Sistem Mimarisi

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

## Pipeline ve İş Akışı

### 1. Register Pipeline (2-Phase)

#### Phase 1: AccountNumber Generation

```
Frontend → POST /api/v1/register (body: {} veya {"confirm": false})
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

## Güvenlik Modeli

### AccountNumber Üretimi

AccountNumber, 192-bit kriptografik güvenlik ile üretilir:

1. `crypto/rand.Read(24 bytes)` → 192-bit entropy
2. `base64.RawURLEncoding.EncodeToString()` → 32 karakter Base64URL
3. Sonuç: 32 karakter (A-Z, a-z, 0-9, -, _)
4. Örnek: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"

### Veritabanında AccountNumber Saklama

AccountNumber ASLA plaintext tutulmaz:

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

Token Bucket algoritması kullanılır:
- Capacity: 20 requests
- Refill rate: 1 request/second
- Applied to: `/api/v1/login`, `/api/v1/register`
- IP-based tracking
- Response: `429 Too Many Requests`

### Log Masking

AccountNumber ve hassas bilgiler loglarda maskelenir:
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

### AAD Yapısı (39 byte)

AAD (Additional Authenticated Data), cross-user data swap saldırılarını önlemek için kullanılır:

```
┌────────┬─────┬──────────────┬──────────────┬───────┬─────────┐
│ MAGIC  │ VER │  USER_TAG    │   TODO_ID    │ FIELD │ PURPOSE │
│ "PXAD" │ 0x01│ SHA256[:16]  │  UUID (16B)  │ 1-3   │   1     │
│ 4 byte │ 1B  │   16 byte    │   16 byte    │  1B   │   1B    │
└────────┴─────┴──────────────┴──────────────┴───────┴─────────┘

FIELD: 0x01=Title, 0x02=Content, 0x03=Tags
PURPOSE: 0x01=Encryption
```

### Şifreli Alanlar

| Alan | Şifreli | Açıklama |
|------|---------|----------|
| Title | Evet | AES-256-GCM + AAD |
| Tags | Evet | Her tag ayrı şifreli |
| Due Date | Hayır | Metadata (tarih) |
| Priority | Hayır | Metadata (enum) |
| Completed | Hayır | Metadata (boolean) |
| AccountNumber | Hayır | Plaintext DB'de tutulmaz; HMAC hash (account_lookup) ve Argon2id hash (account_hash) tutulur |

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

## Kurulum

### Gereksinimler

- Docker & Docker Compose
- Go 1.21+ (development için)
- OpenSSL (setup.sh için)

### Hızlı Başlangıç

```bash
# Clone repository
git clone <repo-url>
cd todo-app-yavuzlar

# Setup (otomatik .env oluşturur)
./setup.sh

# Build ve test (tüm testler çalışır, başarısız olursa servisler başlamaz)
./build.sh

# Erişim
open http://localhost
```

### Environment Variables

`setup.sh` otomatik olarak aşağıdaki environment variable'ları oluşturur:

| Değişken | Açıklama | Üretim |
|----------|----------|--------|
| `ENCRYPTION_KEY` | 32-byte hex encoded AES key (master key) | `openssl rand -hex 32` |
| `JWT_SECRET` | JWT imzalama anahtarı | `openssl rand -hex 32` |
| `ACCOUNT_LOOKUP_PEPPER` | HMAC lookup için pepper (>=32 bytes) | `openssl rand -hex 32` |
| `JWT_EXPIRATION_MINUTES` | JWT token geçerlilik süresi | `15` (default) |
| `ALLOWED_ORIGIN` | CORS allowed origins | `*` (dev) veya specific domain |
| `APP_ENV` | Environment mode | `development` veya `production` |
| `LOG_LEVEL` | Log seviyesi | `debug`, `info`, `warn`, `error` |
| `DB_HOST` | PostgreSQL host | `postgres` (Docker) veya `localhost` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `postgres` |
| `DB_NAME` | Database name | `todos` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_SSL_MODE` | SSL mode | `disable` (dev) veya `require` (prod) |
| `BACKEND_PORT` | Backend server port | `8080` |

### Docker Compose Servisleri

- `postgres`: PostgreSQL 15 (Alpine)
- `backend`: Go backend (Alpine)
- `frontend`: Nginx (Alpine) - SPA + API proxy

---

## API Referansı

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

**Not:** `account_number` response'ta yoktur (güvenlik).

### Todos (JWT Required)

Tüm todo endpoint'leri JWT token gerektirir. Token `Authorization` header'ında `Bearer <token>` formatında gönderilmelidir.

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

## Referanslar

### Kriptografi ve Güvenlik

- **AES-256-GCM**: NIST SP 800-38D - Galois/Counter Mode (GCM) for Block Ciphers
- **Argon2id**: RFC 9106 - Argon2 Memory-Hard Function for Password Hashing and Key Derivation
- **HMAC-SHA256**: RFC 2104 - HMAC: Keyed-Hashing for Message Authentication
- **JWT**: RFC 7519 - JSON Web Token (JWT)
- **Base64URL**: RFC 4648 Section 5 - Base 64 Encoding with URL and Filename Safe Alphabet

### Zero-Trust

- **Zero-Trust Architecture**: NIST SP 800-207 - Zero Trust Architecture

### Teknoloji Stack

- **Go**: https://go.dev/
- **PostgreSQL**: https://www.postgresql.org/
- **Nginx**: https://nginx.org/
- **Docker**: https://www.docker.com/
- **GORM**: https://gorm.io/

### İlham Kaynakları

- **Mullvad VPN**: AccountNumber-only authentication modeli
- **OWASP**: Security best practices
- **NIST**: Cryptographic standards

### İlgili Dokümantasyon

- **Go crypto/rand**: https://pkg.go.dev/crypto/rand
- **Go crypto/aes**: https://pkg.go.dev/crypto/aes
- **Go crypto/hmac**: https://pkg.go.dev/crypto/hmac
- **golang.org/x/crypto/argon2**: https://pkg.go.dev/golang.org/x/crypto/argon2
- **github.com/golang-jwt/jwt/v5**: https://pkg.go.dev/github.com/golang-jwt/jwt/v5

---

**Son Güncelleme**: Ocak 2026
**Versiyon**: 2.0 (AccountNumber-based authentication)
