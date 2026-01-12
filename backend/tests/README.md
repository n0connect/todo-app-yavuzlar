# 🧪 Test Suite

## 📁 Dizin Yapısı

```
backend/tests/
├── auth/                      # Authentication tests (15 test)
├── database/                   # Migration tests (7 test) ⭐ YENİ
├── encryption/                 # Encryption tests (7 test)
├── handlers/                    # Handler tests (2 test)
├── integration/                # Integration tests (5 test)
├── middleware/                 # Middleware tests (15 test)
├── testutil/                   # Test utilities
│   ├── db.go                  # Database setup helpers
│   ├── db_helper.go           # Database env setup ⭐ YENİ
│   └── env.go                 # Environment setup
├── todo/                       # Todo service tests (6 test)
└── utils/                      # Utils tests (3 test)
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
| Encryption | 7 | ✅ |
| Middleware | 15 | ✅ |
| Todo Service | 6 | ✅ |
| Integration | 5 | ⚠️ (2 skip) |
| Handlers | 2 | ✅ |
| Utils | 3 | ✅ |
| **TOPLAM** | **60** | **✅ 58 PASS, 2 SKIP** |

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
- `TestDatabaseEncryptionKey`: Encryption key'i test eder

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
- `JWT_SECRET`, `ENCRYPTION_KEY`, `ACCOUNT_LOOKUP_PEPPER`

## 📚 Dokümantasyon

- `tests/README.md`: Bu dosya
- `tests/database/README.md`: Migration testleri detayları
- `tests/MIGRATION_TEST_GUIDE.md`: Migration test kılavuzu
- `tests/TEST_ANALYSIS.md`: Test analizi ve kapsam raporu
- `tests/TEST_IMPLEMENTATION_SUMMARY.md`: Test implementation özeti
- `tests/FINAL_TEST_REPORT.md`: Final test raporu

---

**Son Güncelleme:** Ocak 2026
