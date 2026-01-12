# 🔐 Todo App - Zero-Trust Encrypted Todo Application

MullvadVPN'den esinlenilmiş, **AccountNumber-only** kimlik doğrulama ve uçtan uca şifreleme ile güvenli todo uygulaması.

---

## 📋 İçindekiler

- [Özellikler](#-özellikler)
- [Sistem Mimarisi](#-sistem-mimarisi)
- [Pipeline ve İş Akışı](#-pipeline-ve-iş-akışı)
- [Güvenlik Modeli](#-güvenlik-modeli)
- [Kurulum](#-kurulum)
- [API Referansı](#-api-referansı)
- [Frontend Yapısı](#-frontend-yapısı)
- [Backend Yapısı](#-backend-yapısı)
- [Test](#-test)

---

## ✨ Özellikler

### Güvenlik
- 🔑 **AccountNumber-Only Auth**: Şifre yok, 32 karakterlik kriptografik AccountNumber (192-bit entropy, Base64URL)
- 🔒 **AES-256-GCM**: Tüm veriler (title + tags) şifreli saklanır
- 🛡️ **AAD Koruması**: Cross-user data swap saldırılarına karşı koruma
- 🔐 **Per-User Keys**: Her kullanıcıya özel şifreleme anahtarı
- 📜 **Strict CSP**: XSS ve injection saldırılarına karşı koruma
- 🔐 **Zero Plaintext Storage**: AccountNumber veritabanında plaintext tutulmaz (HMAC lookup + Argon2id hash)
- 🚦 **Rate Limiting**: Login ve register endpoint'leri için IP bazlı rate limiting
- 🎭 **Log Masking**: AccountNumber ve hassas bilgiler loglarda maskelenir
- 🔒 **Session Storage**: Token'lar localStorage yerine sessionStorage'da (tab kapanınca silinir)

### Fonksiyonel
- ✅ Todo oluşturma, düzenleme, silme
- 🏷️ Etiket (tag) sistemi - şifreli
- 📅 Due date (son tarih)
- ⚡ Öncelik seviyeleri (low/medium/high)
- 🔍 Tag ile filtreleme
- 📊 Sıralama (tarih/öncelik/alfabetik)

### UI/UX
- 👁️ Progressive AccountNumber reveal animasyonu
- 🎨 Minimal, monospace tasarım
- ⚡ Vanilla JS - framework bağımlılığı yok
- 📱 Responsive tasarım

---

## 🏗️ Sistem Mimarisi

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              FRONTEND                                   │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  index.html + styles.css + js/{app,auth,todo,ui,utils,notif}.js │    │
│  │  - sessionStorage (JWT token, AccountNumber)                     │    │
│  │  - Progressive AccountNumber reveal                             │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ HTTP (Same-Origin Proxy)
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           NGINX (Reverse Proxy)                         │
│  - SPA routing (try_files)                                            │
│  - /api/* → backend:8080                                              │
│  - Security headers (CSP, X-Frame-Options, etc.)                      │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         BACKEND (Go net/http)                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    MIDDLEWARE CHAIN                             │    │
│  │  SecurityHeaders → CORS → RateLimit → JWT Auth                │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      HANDLERS                                   │    │
│  │  Register (2-phase) │ Login │ GetTodos │ CreateTodo │ ...     │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    SERVICE LAYER                                │    │
│  │  TodoService: Validation → Encryption → Repository            │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐             │
│  │   Encryption   │  │      Auth      │  │   Repository   │             │
│  │   AES-256-GCM  │  │   JWT (15min)  │  │   GORM/PG      │             │
│  │   + AAD        │  │   Argon2id     │  │                │             │
│  │                │  │   HMAC lookup  │  │                │             │
│  └────────────────┘  └────────────────┘  └────────────────┘             │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         POSTGRESQL                                      │
│  users:                                                                  │
│    - uuid (internal ID, 24 chars)                                       │
│    - account_lookup (HMAC-SHA256(pepper, AccountNumber))                │
│    - account_hash (Argon2id(AccountNumber))                             │
│    - encrypted_key (AES-256-GCM encrypted user key)                     │
│  todos:                                                                  │
│    - id, user_uuid, title(enc), tags(enc), due_date, priority          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Pipeline ve İş Akışı

### 1. Register Pipeline (2-Phase Zero-Trust)

```
┌─────────────────────────────────────────────────────────────────────────┐
│ PHASE 1: AccountNumber Generation                                      │
└─────────────────────────────────────────────────────────────────────────┘

Frontend:
  POST /api/v1/register
  Body: {} (empty) veya { "confirm": false }

  ↓

Backend (RegisterHandler):
  1. GenerateAccountNumber()
     - crypto/rand: 24 bytes (192-bit)
     - base64.RawURLEncoding → 32 chars
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

  ↓

Response:
  {
    "success": true,
    "message": "Account number generated - click continue to create account",
    "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq",
    "pending_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "confirmed": false
  }

  ↓

Frontend:
  - Display AccountNumber to user (progressive reveal)
  - Store pending_token in memory
  - User clicks "Continue" → Phase 2


┌─────────────────────────────────────────────────────────────────────────┐
│ PHASE 2: Account Creation (Zero-Trust)                                 │
└─────────────────────────────────────────────────────────────────────────┘

Frontend:
  POST /api/v1/register
  Body: {
    "confirm": true,
    "pending_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }

  ↓

Backend (RegisterHandler):
  1. VerifyPendingRegistrationToken(pendingToken)
     - Decode JWT → extract pending_id
     - GetPendingRegistration(pending_id)
       → Retrieve AccountNumber from server-side store
       → Delete entry (one-time use)
     - ZERO-TRUST: AccountNumber from token, NOT user input

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

  ↓

Response:
  {
    "success": true,
    "message": "Account created successfully",
    "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "confirmed": true
  }

  ↓

Frontend:
  - Store JWT token in sessionStorage
  - Store AccountNumber in sessionStorage
  - Redirect to app (loadTodos)
```

### 2. Login Pipeline

```
Frontend:
  POST /api/v1/login
  Body: {
    "account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
  }

  ↓

Backend (LoginHandler):
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

  ↓

Response:
  {
    "success": true,
    "message": "Login successful",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    // Note: account_number NOT in response (security)
  }

  ↓

Frontend:
  - Store JWT token in sessionStorage
  - Use existing AccountNumber from input
  - Redirect to app (loadTodos)
```

### 3. Todo CRUD Pipeline

```
┌─────────────────────────────────────────────────────────────────────────┐
│ CREATE TODO                                                             │
└─────────────────────────────────────────────────────────────────────────┘

Frontend:
  POST /api/v1/todos
  Headers: { Authorization: "Bearer <JWT>" }
  Body: {
    "title": "Buy groceries",
    "tags": ["shopping", "urgent"],
    "priority": "high",
    "due_date": "2026-01-15"
  }

  ↓

Backend:
  1. JWTMiddleware
     - Extract JWT from Authorization header
     - Verify signature, expiration, issuer, audience
     - Extract user UUID from subject
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

  ↓

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


┌─────────────────────────────────────────────────────────────────────────┐
│ GET TODOS                                                               │
└─────────────────────────────────────────────────────────────────────────┘

Frontend:
  GET /api/v1/todos
  Headers: { Authorization: "Bearer <JWT>" }

  ↓

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

  ↓

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

### 4. Encryption Pipeline

```
┌─────────────────────────────────────────────────────────────────────────┐
│ ENCRYPTION FLOW                                                          │
└─────────────────────────────────────────────────────────────────────────┘

Layer 1: Master Key (Environment Variable)
  ENCRYPTION_KEY (32 bytes, hex encoded)
  ├── Used to encrypt/decrypt user-specific AES keys
  └── Never stored in database

  ↓

Layer 2: User-Specific AES Key
  GenerateUserAESKey() → 32 bytes
  ├── Encrypted with ENCRYPTION_KEY (AES-256-GCM)
  └── Stored in users.encrypted_key

  ↓

Layer 3: AAD (Additional Authenticated Data)
  BuildAAD(userUUID, todoID, fieldType)
  ├── Format: "PXAD" + version + userTag + todoID + field + purpose
  ├── 39 bytes total
  └── Prevents cross-user data swapping

  ↓

Layer 4: Data Encryption
  EncryptWithAAD(userKey, plaintext, aad)
  ├── AES-256-GCM encryption
  ├── Nonce: 12 bytes (random per encryption)
  └── Output: nonce + ciphertext + tag (16 bytes)

  ↓

Database Storage:
  todos.title: <nonce><ciphertext><tag> (binary)
  todos.tags: JSON array of encrypted strings
```

---

## 🔐 Güvenlik Modeli

### AccountNumber Üretimi

```
GenerateAccountNumber():
  1. crypto/rand.Read(24 bytes)  → 192-bit entropy
  2. base64.RawURLEncoding.EncodeToString()
  3. Result: 32 characters (Base64URL charset)
  4. Example: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
```

### Veritabanında AccountNumber Saklama

**AccountNumber ASLA plaintext tutulmaz:**

```
AccountNumber: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
  ↓
HMAC-SHA256(ACCOUNT_LOOKUP_PEPPER, AccountNumber)
  ↓
account_lookup: "b37e5a5c1f4e510a..." (64 hex chars)
  ├── Fast lookup (indexed)
  └── One-way function (cannot reverse)

AccountNumber: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
  ↓
Argon2id(AccountNumber)
  ├── Memory: 64MB
  ├── Time: 3
  ├── Parallelism: 2
  └── Format: "$argon2id$v=19$m=67108864,t=3,p=2$salt$hash"
  ↓
account_hash: "$argon2id$v=19$m=67108864,t=3,p=2$..." (stored)
  ├── Verification only
  └── Cannot reverse
```

### Rate Limiting

```
Token Bucket Algorithm:
  - Capacity: 20 requests
  - Refill rate: 1 request/second
  - Applied to: /api/v1/login, /api/v1/register
  - IP-based tracking
  - Response: 429 Too Many Requests
```

### Log Masking

```
AccountNumber: "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
  ↓
MaskAccountNumber()
  ↓
Log output: "****************************ATTDq"
  ├── Last 4 characters visible
  └── Rest masked with asterisks
```

### AAD Yapısı (39 byte)

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
| Title | ✅ | AES-256-GCM + AAD |
| Tags | ✅ | Her tag ayrı şifreli |
| Due Date | ❌ | Metadata (tarih) |
| Priority | ❌ | Metadata (enum) |
| Completed | ❌ | Metadata (boolean) |
| AccountNumber | ❌ | **ASLA DB'de tutulmaz** |

### Content Security Policy

```
default-src 'self';
script-src 'self';                    # No inline scripts
style-src 'self' 'unsafe-inline';     # Inline styles for dynamic UI
font-src 'self' https://fonts.gstatic.com;
connect-src 'self';
frame-ancestors 'none';               # No framing (clickjacking)
object-src 'none';                    # No plugins
```

---

## 🚀 Kurulum

### Gereksinimler

- Docker & Docker Compose
- Go 1.21+ (development için)
- OpenSSL (setup.sh için)

### Hızlı Başlangıç

```bash
# Clone
git clone <repo-url>
cd todo-app-yavuzlar

# Setup (otomatik .env oluşturur)
./setup.sh

# Environment variables otomatik oluşturulur:
# - ENCRYPTION_KEY (32 bytes, hex)
# - JWT_SECRET (32 bytes, hex)
# - ACCOUNT_LOOKUP_PEPPER (32 bytes, hex)
# - JWT_EXPIRATION_MINUTES (default: 15)
# - ALLOWED_ORIGIN (default: *)
# - APP_ENV (default: development)
# - LOG_LEVEL (default: debug)

# Erişim
open http://localhost
```

### Environment Variables

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

```yaml
services:
  postgres:    # PostgreSQL 15 (Alpine)
  backend:   # Go backend (Alpine)
  frontend:   # Nginx (Alpine) - SPA + API proxy
```

---

## 📡 API Referansı

### Authentication

#### Register (Phase 1: Generate AccountNumber)

```http
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

```http
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

```http
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

**Note:** `account_number` response'ta **YOK** (güvenlik).

### Todos (JWT Required)

#### Get Todos

```http
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
    "created_at": "2026-01-12T10:00:00Z",
    "updated_at": "2026-01-12T10:00:00Z"
  }
]
```

#### Create Todo

```http
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
  "created_at": "2026-01-12T10:00:00Z",
  "updated_at": "2026-01-12T10:00:00Z"
}
```

#### Update Todo

```http
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

#### Delete Todo

```http
DELETE /api/v1/todos/{id}
Authorization: Bearer <JWT>
```

**Response:**
```json
{
  "success": true,
  "message": "Todo deleted successfully"
}
```

### Error Responses

```json
{
  "success": false,
  "message": "Error message"
}
```

**Status Codes:**
- `200 OK`: Success
- `201 Created`: Resource created
- `400 Bad Request`: Invalid request
- `401 Unauthorized`: Invalid credentials or expired token
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

---

## 🎨 Frontend Yapısı

```
frontend/
├── index.html              # Ana HTML (SPA)
├── styles.css              # Global stiller
├── eye-hover-animation.css # Göz animasyon stilleri
├── secure-hover-animation.css # Güvenlik animasyon stilleri
├── uuid-animation.css      # AccountNumber reveal animasyonu
├── eye-hover-animation.js  # Progressive reveal sistemi
├── secure-hover-animation.js # Güvenlik animasyonu
├── validation.js           # Frontend validation
└── js/
    ├── app.js              # Event listeners, init, routing
    ├── auth.js             # Login/Register logic (2-phase)
    ├── todo.js             # CRUD operations
    ├── ui.js               # Render functions
    ├── utils.js            # Helpers (API calls, token management)
    └── notifications.js    # Toast notifications
```

### Progressive AccountNumber Reveal

Göz ikonu etrafında 3 katmanlı animasyon sistemi:

| Zone | Yarıçap | Davranış |
|------|---------|----------|
| Outer | 48-87px | Yavaş rastgele karakter scramble |
| Middle | 18-51px | 2x hızda yoğun scramble |
| Inner | 0-18px | Smooth reveal - gerçek AccountNumber görünür |

### Storage Strategy

- **sessionStorage**: JWT token ve AccountNumber (tab kapanınca silinir)
- **localStorage**: Kullanılmıyor (güvenlik)

---

## 📁 Backend Yapısı

```
backend/
├── cmd/
│   └── server/
│       └── main.go         # Entry point, route setup
└── internal/
    ├── auth/               # Authentication & JWT
    │   ├── account.go      # AccountNumber hashing (Argon2id)
    │   ├── context.go      # User context from JWT
    │   ├── jwt.go          # JWT signing/verification
    │   └── pending_store.go # In-memory pending registration store
    ├── config/             # Environment configuration
    │   └── config.go       # Config loading & validation
    ├── database/           # Database connection & migration
    │   └── connection.go   # GORM setup, AutoMigrate
    ├── encryption/         # AES-256-GCM encryption
    │   ├── crypto.go       # Encryption/decryption
    │   └── aad.go          # AAD builder
    ├── handlers/           # HTTP handlers
    │   ├── auth.go         # Register, Login
    │   └── todos.go        # Todo CRUD
    ├── middleware/         # HTTP middleware
    │   ├── auth.go         # JWT authentication
    │   ├── cors.go         # CORS handling
    │   ├── ratelimit.go    # Rate limiting (token bucket)
    │   ├── security.go     # Security headers
    ├── models/             # Data models
    │   ├── requests.go     # Request/Response DTOs
    │   ├── todo.go         # Todo model
    │   └── user.go         # User model
    ├── store/              # Repository layer
    │   ├── todo.go         # Todo repository
    │   └── user.go         # User repository
    ├── todo/               # Business logic
    │   └── service.go      # Todo service (encryption layer)
    └── utils/              # Utilities
        ├── base64.go       # Base64 helpers
        ├── json.go         # JSON encoding/decoding
        ├── logger.go       # Structured logging
        └── validation.go  # AccountNumber generation & validation
```

### Middleware Chain

```
Request → SecurityHeadersMiddleware
        → CORSMiddleware
        → RateLimitMiddleware (login/register only)
        → JWTMiddleware (protected routes only)
        → Handler
```

---

## 🧪 Test

### Test Dosyaları

```
backend/
├── internal/
│   ├── auth/
│   │   ├── account_test.go           # Argon2id & HMAC tests
│   │   ├── jwt_pending_test.go       # Pending token tests
│   │   └── pending_store_test.go     # Pending store tests
│   ├── handlers/
│   │   └── auth_test.go              # Handler response tests
│   └── utils/
│       └── validation_test.go        # AccountNumber tests
```

### Test Çalıştırma

```bash
# Tüm testler
cd backend
go test ./...

# Sadece auth testleri
go test ./internal/auth -v

# Sadece pending store testleri
go test ./internal/auth -v -run TestPending

# Race detector ile
go test -race ./...
```

### Test Kapsamı

- ✅ AccountNumber generation (192-bit, Base64URL)
- ✅ AccountNumber validation (regex, length)
- ✅ HMAC lookup computation
- ✅ Argon2id hashing & verification
- ✅ Pending registration store (store, get, delete, expiration)
- ✅ Pending token (pending_id, one-time use)
- ✅ Login response (no account_number)
- ✅ Register response structure

---

## 📜 Lisans

MIT License

---

**Son Güncelleme**: Ocak 2026

**Versiyon**: 2.0 (AccountNumber-based authentication)
