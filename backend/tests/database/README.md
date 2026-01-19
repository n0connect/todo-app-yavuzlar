# 🗄️ Database Migration Tests

## 📋 Amaç

Bu testler **backend ayağa kalkmadan önce** çalıştırılmalıdır. Amaçları:

1. ✅ Database bağlantısının doğru çalıştığını doğrulamak
2. ✅ Migration'ların başarıyla çalıştığını test etmek
3. ✅ Database schema'sının doğru oluşturulduğunu doğrulamak
4. ✅ Index'lerin ve constraint'lerin doğru ayarlandığını test etmek
5. ✅ Database'in backend başlamadan önce hazır olduğunu garantilemek

## 🚀 Kullanım

### Test Database Setup

Testler otomatik olarak test database'ine bağlanır. Environment variables:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=todos_test
export DB_SSL_MODE=disable
```

### Test Çalıştırma

```bash
# Tüm migration testleri
cd backend
go test ./tests/database -v

# Sadece migration testi
go test ./tests/database -v -run TestDatabaseMigration

# Schema testleri
go test ./tests/database -v -run TestUserTableSchema
go test ./tests/database -v -run TestTodoTableSchema

# Connection testi
go test ./tests/database -v -run TestDatabaseConnection
```

### Backend Başlatmadan Önce

```bash
# 1. Migration testlerini çalıştır
go test ./tests/database -v

# 2. Testler başarılıysa backend'i başlat
go run ./cmd/server
```

## 📝 Test Kategorileri

### 1. Migration Tests

- `TestDatabaseMigration`: Migration'ların başarıyla çalıştığını test eder
- `TestDatabaseConnection`: Database bağlantısını test eder
- `TestDatabaseMasterKey`: Master key'in initialize edildiğini test eder

### 2. Schema Tests

- `TestUserTableSchema`: User tablosunun doğru schema'ya sahip olduğunu test eder
- `TestTodoTableSchema`: Todo tablosunun doğru schema'ya sahip olduğunu test eder

### 3. Constraint Tests

- `TestUserTableIndexes`: User tablosundaki index'leri test eder
- `TestDatabaseConstraints`: NOT NULL constraint'lerini test eder

## ⚠️ Önemli Notlar

1. **Test Database**: Testler `todos_test` database'ini kullanır (production database'i değil)
2. **Migration**: `database.Init()` çağrıldığında migration'lar otomatik çalışır
3. **Clean State**: Her test öncesi database temizlenmez (manuel cleanup gerekebilir)
4. **Production**: Production'da bu testler çalıştırılmamalı

## 🔧 Test Infrastructure

Testler `tests/testutil/db.go` içindeki utility fonksiyonlarını kullanır:

- `SetupTestDB()`: Test database bağlantısı kurar
- `TeardownTestDB()`: Test sonrası temizlik
- `CleanTestDatabase()`: Database'i temizler ve migration'ları yeniden çalıştırır
- `InitializeTestDatabase()`: Global database connection'ı initialize eder

---

**Son Güncelleme:** Ocak 2026
