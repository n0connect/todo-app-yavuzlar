# Not Simple Todo App - Encrypted Todo Application

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![JavaScript](https://img.shields.io/badge/JavaScript-ES6+-F7DF1E?style=flat&logo=javascript)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat&logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)
![Nginx](https://img.shields.io/badge/Nginx-Alpine-009639?style=flat&logo=nginx)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

Secure todo application with Bearer CSPRNG token-based AccountNumber-only authentication and per-user encryption.

<p align="center">
  <img src="frontend/images/mainpage.png" alt="Main Page" width="48%" />
  <img src="frontend/images/registerpage.png" alt="Register Page" width="48%" />
</p>

---

## Overview

This application is a todo management system that authenticates using only cryptographic AccountNumber (no passwords) and encrypts all data with per-user encryption.

## Summary / Update

- OpenSSL-backed crypto engine for AES-256-GCM, HMAC-SHA256, SHA-256, Argon2id, and CSPRNG (JWT stays native)
- `/api/v2` only: 256-bit AccountNumber + AAD v2 with full SHA-256 user tag
- Master key rotation with active key ID + explicit old key list (decrypt-only, no silent fallback; legacy ENCRYPTION_KEY removed)
- Config-driven limits with secure defaults in config/setup; override via `.env`

## Summary / Implementations

- OpenSSL-backed crypto engine for AES-256-GCM, HMAC-SHA256, SHA-256, Argon2id, and CSPRNG
- Master key rotation with `MASTER_KEY_ACTIVE_ID` + `MASTER_KEY_ACTIVE` and optional `MASTER_KEY_OLD`
- `/api/v2` endpoints with 256-bit AccountNumber (43 chars, Base64URL) and AAD v2 (full SHA-256 user tag)
- Zero-trust request handling with strict validation and generic client errors

### Key Features

**Security:**
- AccountNumber-only authentication (256-bit/43 chars)
- AES-256-GCM encryption (title and tags encrypted)
- AAD (Additional Authenticated Data) for cross-user data swap protection
- Per-user encryption keys (unique key per user)
- Zero plaintext storage (AccountNumber never stored in plaintext in database)
- Rate limiting (IP-based, for login/register)
- Log masking (AccountNumber and sensitive information masked)
- Session storage (tokens in sessionStorage instead of localStorage)
- Trusted proxy-aware client IP extraction for security decisions

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

## Security Model

### AccountNumber Generation

AccountNumber is generated with CSPRNG and Base64URL (no padding):

**Current (v2):**
1. `OpenSSL RAND_bytes(32 bytes)` → 256-bit entropy
2. `base64.RawURLEncoding.EncodeToString()` → 43 characters Base64URL
3. Result: 43 characters (A-Z, a-z, 0-9, -, _)

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
  ├── Memory: ARGON2_MEMORY_KIB
  ├── Time: ARGON2_TIME
  ├── Parallelism: ARGON2_PARALLELISM
  └── Format: "$argon2id$v=19$m=<memory>,t=<time>,p=<parallelism>$salt$hash"
    │
    ▼
account_hash: "$argon2id$v=19$m=<memory>,t=<time>,p=<parallelism>$..." (stored)
  ├── Verification only
  └── Cannot reverse
```

### Rate Limiting

Token Bucket algorithm is used (config-driven):
- Capacity: `RATE_LIMIT_MAX_TOKENS`
- Refill rate: `RATE_LIMIT_REFILL_INTERVAL_SEC`
- Applied to: `/api/v2/login`, `/api/v2/register`
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

### Encryption Architecture

Data encryption uses a multi-layer approach:

1. **Master Key** (`MASTER_KEY_ACTIVE` + `MASTER_KEY_ACTIVE_ID`): 32-byte AES key stored in environment variables, used to encrypt/decrypt user-specific keys (old keys in `MASTER_KEY_OLD`)
2. **User-Specific AES Key**: 32-byte AES-256 key per user, encrypted with master key and stored in `users.encrypted_key`
3. **AAD (Additional Authenticated Data)**: v2 55-byte structure that prevents cross-user data swap attacks
4. **Data Encryption**: AES-256-GCM encryption with 12-byte nonce per encryption

### AAD Structure

AAD (Additional Authenticated Data) is used to prevent cross-user data swap attacks:

**v2 (55 bytes):**
`MAGIC(4) || VER(1) || USER_TAG(32) || TODO_ID(16) || FIELD(1) || PURPOSE(1)`

```
┌────────┬─────┬────────────────────────┬──────────────┬───────┬─────────┐
│ MAGIC  │ VER │        USER_TAG        │   TODO_ID    │ FIELD │ PURPOSE │
│ "PXAD" │ 0x02│ SHA256(userUUID, 32B)  │  UUID (16B)  │ 1-3   │   1     │
│ 4 byte │ 1B  │         32 byte        │   16 byte    │  1B   │   1B    │
└────────┴─────┴────────────────────────┴──────────────┴───────┴─────────┘

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
| AccountNumber | Not | Plaintext not stored in DB; HMAC hash (account_lookup) and Argon2id hash (account_hash) stored |

## Installation

### Requirements

- Docker & Docker Compose
- Go 1.21+ (for development)
- OpenSSL >= 3.2.0 (for setup.sh and OpenSSL-based crypto)
- OpenSSL 3.2 dev headers + pkg-config (for local builds/tests)

### Quick Start

```bash
# Configure .env values, generate secrets, then build/test
./setup.sh
./build.sh --clean
```

### Environment Variables

`setup.sh` generates secrets and applies secure defaults for non-secret values (override in `.env`).

| Variable | Description | Default | Notes |
|----------|-------------|---------|-------|
| `JWT_SECRET` | JWT signing key | Generated | 32+ chars (OpenSSL rand recommended) |
| `JWT_SECRET_MIN_LEN` | Minimum JWT secret length | `32` | Integer |
| `JWT_ISSUER` | JWT issuer | `todo-app-backend` | Override for your environment |
| `JWT_AUDIENCE` | JWT audience | `todo-app-frontend` | Override for your environment |
| `JWT_EXPIRATION_MINUTES` | JWT token validity period | `15` | Integer |
| `MASTER_KEY_ACTIVE_ID` | Active master key ID | Generated | Required |
| `MASTER_KEY_ACTIVE` | Active master key | Generated | 32 bytes hex or raw |
| `MASTER_KEY_OLD` | Optional old keys | Empty | `kid:hex` comma-separated |
| `ACCOUNT_LOOKUP_PEPPER` | HMAC pepper for account lookup | Generated | >=32 bytes (hex/base64/raw) |
| `ALLOWED_ORIGINS` | CORS allowed origins | `http://localhost` | Comma-separated; no `*` in prod |
| `TRUSTED_PROXIES` | Trusted proxy IPs/CIDRs | Empty | Optional |
| `DB_HOST` | PostgreSQL host | `localhost` | Override in production |
| `DB_PORT` | PostgreSQL port | `5432` | Override in production |
| `DB_USER` | PostgreSQL user | `postgres` | Override in production |
| `DB_PASSWORD` | PostgreSQL password | `postgres` | Override in production |
| `DB_NAME` | PostgreSQL database name | `todo_app` | Override in production |
| `DB_SSL_MODE` | SSL mode | `disable` | Must not be `disable` in prod |
| `BACKEND_PORT` | Backend server port | `8080` | Required |
| `MAX_BASE64_LOGIN_LEN` | Base64 request size limit (login) | `4096` | Integer |
| `MAX_BASE64_TODO_LEN` | Base64 request size limit (todo) | `65536` | Integer |
| `MAX_REQUEST_BODY_BYTES` | HTTP request body size limit | `1048576` | Integer |
| `MAX_JSON_DEPTH` | Max JSON nesting depth | `2` | Integer |
| `MAX_TITLE_LENGTH` | Max todo title length | `1000` | Integer |
| `MIN_TITLE_LENGTH` | Min todo title length | `1` | Integer |
| `MAX_TAG_LENGTH` | Max tag length | `6` | Integer |
| `MAX_TAGS_PER_TODO` | Max tags per todo | `10` | Integer |
| `ARGON2_MEMORY_KIB` | Argon2 memory cost (KiB) | `65536` | Integer |
| `ARGON2_TIME` | Argon2 time cost | `3` | Integer |
| `ARGON2_PARALLELISM` | Argon2 parallelism | `2` | Integer |
| `ARGON2_SALT_LENGTH` | Argon2 salt length (bytes) | `16` | Integer |
| `ARGON2_HASH_LENGTH` | Argon2 hash length (bytes) | `32` | Integer |
| `PENDING_TOKEN_TTL_SEC` | Pending token TTL (seconds) | `300` | Integer |
| `PENDING_CLEANUP_INTERVAL_SEC` | Pending store cleanup interval | `60` | Integer |
| `PENDING_ID_BYTES` | Pending ID size (bytes) | `16` | Integer |
| `INTERNAL_ID_LENGTH` | Internal user ID length | `24` | Integer |
| `RATE_LIMIT_MAX_TOKENS` | Rate limit bucket size | `20` | Integer |
| `RATE_LIMIT_REFILL_INTERVAL_SEC` | Rate limit refill interval | `3` | Integer |
| `RATE_LIMIT_CLEANUP_INTERVAL_SEC` | Rate limit cleanup interval | `300` | Integer |
| `RATE_LIMIT_MAX_BUCKETS` | Rate limit bucket cap | `10000` | Integer |
| `SERVER_READ_HEADER_TIMEOUT_SEC` | Server read header timeout | `5` | Integer |
| `SERVER_READ_TIMEOUT_SEC` | Server read timeout | `15` | Integer |
| `SERVER_WRITE_TIMEOUT_SEC` | Server write timeout | `15` | Integer |
| `SERVER_IDLE_TIMEOUT_SEC` | Server idle timeout | `60` | Integer |
| `SERVER_MAX_HEADER_BYTES` | Server max header bytes | `8192` | Integer |
| `HSTS_MAX_AGE` | HSTS max-age (seconds) | `0` | Optional |
| `HSTS_INCLUDE_SUBDOMAINS` | HSTS includeSubDomains | `false` | Optional |
| `HSTS_PRELOAD` | HSTS preload | `false` | Optional |
| `SEC_HEADER_COOP` | Cross-Origin-Opener-Policy | Empty | Optional |
| `SEC_HEADER_CORP` | Cross-Origin-Resource-Policy | Empty | Optional |
| `SEC_HEADER_COEP` | Cross-Origin-Embedder-Policy | Empty | Optional |
| `APP_ENV` | Environment mode | `development` | `development` or `production` |
| `LOG_LEVEL` | Log level | `info` | `debug`, `info`, `warn`, `error` |

Note: Legacy `ENCRYPTION_KEY` fallback has been removed. Use `MASTER_KEY_ACTIVE` and optional `MASTER_KEY_OLD` for rotation and migration.

## API Reference

### JWT Authentication
Protected endpoints require `Authorization: Bearer <token>` header. Token expiration is controlled by `JWT_EXPIRATION_MINUTES`.

### Endpoints

- `POST /api/v2/register` - Generate AccountNumber (Phase 1) or create account (Phase 2)
- `POST /api/v2/login` - Login with AccountNumber
- `GET /api/v2/todos` - Get all todos (JWT required)
- `POST /api/v2/todos` - Create todo (JWT required)
- `PUT /api/v2/todos/{id}` - Update todo (JWT required)
- `DELETE /api/v2/todos/{id}` - Delete todo (JWT required)

### Error Codes

- `200` - Success
- `201` - Created
- `204` - No Content
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error

---

## References

- **AES-256-GCM**: NIST SP 800-38D - Galois/Counter Mode (GCM) for Block Ciphers
- **Argon2id**: RFC 9106 - Argon2 Memory-Hard Function for Password Hashing and Key Derivation
- **HMAC-SHA256**: RFC 2104 - HMAC: Keyed-Hashing for Message Authentication
- **JWT**: RFC 7519 - JSON Web Token (JWT)
- **Base64URL**: RFC 4648 Section 5 - Base 64 Encoding with URL and Filename Safe Alphabet
- **UUID**: RFC 4122 - A Universally Unique IDentifier (UUID) URN Namespace
- **Token Bucket Algorithm**: Rate limiting algorithm for traffic shaping
- **Zero-Trust Architecture**: NIST SP 800-207 - Zero Trust Architecture
- **Go**: https://go.dev/
- **PostgreSQL**: https://www.postgresql.org/
- **Nginx**: https://nginx.org/
- **Docker**: https://www.docker.com/
- **GORM**: https://gorm.io/
- **Web Storage API**: https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API (SessionStorage)
- **CORS**: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
- **Mullvad VPN**: AccountNumber-only authentication model
- **OWASP**: Security best practices
- **NIST**: Cryptographic standard's
- **OpenSSL 3**: https://www.openssl.org/docs/man3.0/
- **github.com/golang-jwt/jwt/v5**: https://pkg.go.dev/github.com/golang-jwt/jwt/v5
- **github.com/google/uuid**: https://pkg.go.dev/github.com/google/uuid
- **gorm.io/driver/postgres**: https://pkg.go.dev/gorm.io/driver/postgres

---

<div align="center">
  <img src="frontend/images/c6acc3600c274500faddcc7f376d90a3faeb1f80cbe0dd119f29da9154462f54.png" alt="How we turned a simple todo app into this" width="400" />
  
  <p><em>How we turned a simple todo app into this</em></p>
</div> 

---

**Last Updated**: January 2026
**Version**: 2.0 (AccountNumber-based authentication)
**Status**: It is an experimental project that has not been completed.
