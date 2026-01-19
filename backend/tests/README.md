# 🧪 Test Suite

## 📁 Dizin Yapısı

```
backend/tests/
├── auth/                      # Authentication tests (15 test)
├── database/                   # Migration tests (7 test) ⭐ YENİ
├── encryption/                 # Encryption tests (10 test)
├── handlers/                    # Handler tests (2 test)
├── integration/                # Integration tests (20 test)
├── middleware/                 # Middleware tests (17 test)
├── testutil/                   # Test utilities
│   ├── db.go                  # Database setup helpers
│   ├── db_helper.go           # Database env setup ⭐ YENİ
│   └── env.go                 # Environment setup
├── todo/                       # Todo service tests (7 test)
└── utils/                      # Utils tests (17 test)
```

## 🚀 Test Çalıştırma Sırası

### ⭐ ÖNEMLİ: Migration Testleri Önce Çalıştırılmalı

Migration testleri **backend ayağa kalkmadan önce** çalıştırılmalıdır:

```bash
# 1. Migration testleri (database hazırlığı)
cd backend
go test ./tests/database -v

# 2. Testler başarılıysa backend'i başlat
go run ./cmd/server
```

### Tüm Testler

```bash
# Migration testleri (öncelikli)
go test ./tests/database -v

# Diğer testler
go test ./tests/... -v
```

### Kategori Bazlı

```bash
# Migration tests (database hazırlığı)
go test ./tests/database -v

# Authentication tests
go test ./tests/auth -v

# Encryption tests
go test ./tests/encryption -v

# Middleware tests
go test ./tests/middleware -v

# Integration tests
go test ./tests/integration -v

# Todo service tests
go test ./tests/todo -v

# Utils tests
go test ./tests/utils -v
```

## 📊 Test İstatistikleri

| Kategori | Test Sayısı | Durum |
|----------|-------------|-------|
| **Database Migration** | **7** | ⭐ YENİ |
| Authentication | 15 | ✅ |
| Encryption | 10 | ✅ |
| Middleware | 17 | ✅ |
| Todo Service | 7 | ✅ |
| Integration | 20 | ✅ |
| Handlers | 2 | ✅ |
| Utils | 17 | ✅ |
| **TOPLAM** | **95** | ✅ |

## 🗄️ Migration Testleri

### Amaç

Migration testleri backend başlamadan önce:
1. ✅ Database bağlantısını test eder
2. ✅ Migration'ların başarıyla çalıştığını doğrular
3. ✅ Database schema'sının doğru oluşturulduğunu test eder
4. ✅ Index'lerin ve constraint'lerin doğru ayarlandığını doğrular
5. ✅ **Database'in backend başlamadan önce hazır olduğunu garantiler**

### Testler

- `TestDatabaseMigration`: Migration'ların çalıştığını test eder
- `TestUserTableSchema`: User tablosu schema'sını test eder
- `TestTodoTableSchema`: Todo tablosu schema'sını test eder
- `TestUserTableIndexes`: Index'leri test eder
- `TestDatabaseConstraints`: Constraint'leri test eder
- `TestDatabaseConnection`: Bağlantıyı test eder
- `TestDatabaseMasterKey`: Master key'i test eder

### Detaylı Bilgi

Bkz: `tests/database/README.md` ve `tests/MIGRATION_TEST_GUIDE.md`

## 📝 Test Kategorileri

### 1. Unit Tests

**Lokasyon:** `tests/auth/`, `tests/encryption/`, `tests/utils/`, `tests/todo/`

**Özellikler:**
- Hızlı çalışır
- Database gerektirmez (çoğu)
- İzole edilmiş testler

### 2. Migration Tests ⭐ YENİ

**Lokasyon:** `tests/database/`

**Özellikler:**
- Database gerektirir
- Backend başlamadan önce çalıştırılmalı
- Database'in hazır olduğunu garantiler

### 3. Middleware Tests

**Lokasyon:** `tests/middleware/`

**Testler:**
- JWT authentication
- Rate limiting
- CORS policy
- Security headers

### 4. Integration Tests

**Lokasyon:** `tests/integration/`

**Testler:**
- HTTP handler end-to-end flow
- Middleware chain
- Request/response validation

**Not:** Bazı testler database gerektirir (skip edilir)

## 🔧 Test Infrastructure

### Test Environment Setup

`tests/testutil/env.go`:
- `SetupTestEnv()`: Test environment variables
- `TeardownTestEnv()`: Cleanup

### Database Setup

`tests/testutil/db.go`:
- `SetupTestDB()`: Test database connection
- `TeardownTestDB()`: Database cleanup
- `CleanTestDatabase()`: Clean database state
- `InitializeTestDatabase()`: Global database init

`tests/testutil/db_helper.go` ⭐ YENİ:
- `SetupTestDatabaseEnv()`: Database environment variables

## ⚠️ Notlar

### Database Bağımlılığı

Bazı testler database bağlantısı gerektirir:
- Migration tests: ✅ Database gerektirir
- Integration tests: ⚠️ Bazıları database gerektirir (skip edilir)
- JWT middleware test: ⚠️ Database gerektirir (skip edilir)

### Environment Variables

Testler için gerekli environment variables:
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE`
- `JWT_SECRET`, `JWT_SECRET_MIN_LEN`, `JWT_ISSUER`, `JWT_AUDIENCE`, `JWT_EXPIRATION_MINUTES`
- `MASTER_KEY_ACTIVE`, `MASTER_KEY_ACTIVE_ID`, `MASTER_KEY_OLD`, `ACCOUNT_LOOKUP_PEPPER`
- `ALLOWED_ORIGINS`, `BACKEND_PORT`, `APP_ENV`, `LOG_LEVEL`
- `MAX_BASE64_LOGIN_LEN`, `MAX_BASE64_TODO_LEN`, `MAX_REQUEST_BODY_BYTES`, `MAX_JSON_DEPTH`
- `MAX_TITLE_LENGTH`, `MIN_TITLE_LENGTH`, `MAX_TAG_LENGTH`, `MAX_TAGS_PER_TODO`
- `ARGON2_MEMORY_KIB`, `ARGON2_TIME`, `ARGON2_PARALLELISM`, `ARGON2_SALT_LENGTH`, `ARGON2_HASH_LENGTH`
- `PENDING_TOKEN_TTL_SEC`, `PENDING_CLEANUP_INTERVAL_SEC`, `PENDING_ID_BYTES`, `INTERNAL_ID_LENGTH`
- `RATE_LIMIT_MAX_TOKENS`, `RATE_LIMIT_REFILL_INTERVAL_SEC`, `RATE_LIMIT_CLEANUP_INTERVAL_SEC`, `RATE_LIMIT_MAX_BUCKETS`
- `SERVER_READ_HEADER_TIMEOUT_SEC`, `SERVER_READ_TIMEOUT_SEC`, `SERVER_WRITE_TIMEOUT_SEC`, `SERVER_IDLE_TIMEOUT_SEC`, `SERVER_MAX_HEADER_BYTES`

## 📚 Dokümantasyon

- `tests/README.md`: Bu dosya
- `tests/database/README.md`: Migration testleri detayları
- `tests/MIGRATION_TEST_GUIDE.md`: Migration test kılavuzu
- `tests/TEST_ANALYSIS.md`: Test analizi ve kapsam raporu
- `tests/TEST_IMPLEMENTATION_SUMMARY.md`: Test implementation özeti
- `tests/FINAL_TEST_REPORT.md`: Final test raporu

---

**Son Güncelleme:** Ocak 2026
