# 🔐 Todo App - Zero-Trust Encrypted Todo Application

MullvadVPN'den esinlenilmiş, UUID-only kimlik doğrulama ve uçtan uca şifreleme ile güvenli todo uygulaması.

---

## 📋 İçindekiler

- [Özellikler](#-özellikler)
- [Sistem Mimarisi](#-sistem-mimarisi)
- [Güvenlik Modeli](#-güvenlik-modeli)
- [Kurulum](#-kurulum)
- [API Referansı](#-api-referansı)
- [Frontend Yapısı](#-frontend-yapısı)

---

## ✨ Özellikler

### Güvenlik
- 🔑 **UUID-Only Auth**: Şifre yok, 24 karakterlik kriptografik UUID
- 🔒 **AES-256-GCM**: Tüm veriler (title + tags) şifreli saklanır
- 🛡️ **AAD Koruması**: Cross-user data swap saldırılarına karşı koruma
- 🔐 **Per-User Keys**: Her kullanıcıya özel şifreleme anahtarı
- 📜 **Strict CSP**: XSS ve injection saldırılarına karşı koruma

### Fonksiyonel
- ✅ Todo oluşturma, düzenleme, silme
- 🏷️ Etiket (tag) sistemi - şifreli
- 📅 Due date (son tarih)
- ⚡ Öncelik seviyeleri (low/medium/high)
- 🔍 Tag ile filtreleme
- 📊 Sıralama (tarih/öncelik/alfabetik)

### UI/UX
- 👁️ Progressive UUID reveal animasyonu
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
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ HTTPS (JWT Bearer Token)
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           NGINX (Reverse Proxy)                         │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         BACKEND (Go net/http)                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    MIDDLEWARE CHAIN                             │    │
│  │  SecurityHeaders → CORS → JWT Auth                              │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      HANDLERS                                   │    │
│  │  Register │ Login │ GetTodos │ CreateTodo │ UpdateTodo │ Delete │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    SERVICE LAYER                                │    │
│  │  TodoService: Validation → Encryption → Repository              │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐             │
│  │   Encryption   │  │      Auth      │  │   Repository   │             │
│  │   AES-256-GCM  │  │   JWT (15min)  │  │   GORM/PG      │             │
│  │   + AAD        │  │                │  │                │             │
│  └────────────────┘  └────────────────┘  └────────────────┘             │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         POSTGRESQL                                      │
│  users: uuid, encrypted_key                                             │
│  todos: id, user_uuid, title(enc), tags(enc), due_date, priority        │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🔐 Güvenlik Modeli

### Kimlik Doğrulama Akışı

```
REGISTER:
  1. Backend CSPRNG ile 24-char UUID üretir (143-bit entropy)
  2. Kullanıcıya özel AES-256 key üretilir
  3. AES key, master key ile şifrelenerek DB'de saklanır
  4. UUID kullanıcıya gösterilir (tek seferlik!)

LOGIN:
  1. Kullanıcı UUID girer
  2. Backend UUID'yi doğrular
  3. JWT token üretilir (15dk geçerlilik)
  4. Tüm API istekleri JWT ile korunur
```

### Şifreleme Katmanları

```
┌─────────────────────────────────────────────────────────────┐
│  Layer 1: Master Key (Environment Variable)                 │
│  ├── 32-byte AES-256 key                                    │
│  └── Tüm user key'leri bu key ile şifrelenir                │
├─────────────────────────────────────────────────────────────┤
│  Layer 2: User-Specific AES Key                             │
│  ├── Her kullanıcının kendine özel 32-byte key'i            │
│  └── Master key ile şifrelenip DB'de saklanır               │
├─────────────────────────────────────────────────────────────┤
│  Layer 3: AAD (Additional Authenticated Data)               │
│  ├── Her şifreli veri user+todo+field bilgisiyle bağlanır   │
│  └── Cross-user data swapping saldırılarını engeller        │
└─────────────────────────────────────────────────────────────┘
```

### AAD Yapısı (39 byte)

```
┌────────┬─────┬──────────────┬──────────────┬───────┬─────────┐
│ MAGIC  │ VER │  USER_TAG    │   TODO_ID    │ FIELD │ PURPOSE │
│ "PXAD" │ 0x01│ SHA256[:16]  │  UUID (16B)  │ 1-3   │   1     │
│ 4 byte │ 1B  │   16 byte    │   16 byte    │  1B   │   1B    │
└────────┴─────┴──────────────┴──────────────┴───────┴─────────┘

FIELD: 0x01=Title, 0x02=Content, 0x03=Tags
```

### Şifreli Alanlar

| Alan | Şifreli | Açıklama |
|------|---------|----------|
| Title | ✅ | AES-256-GCM + AAD |
| Tags | ✅ | Her tag ayrı şifreli |
| Due Date | ❌ | Metadata (tarih) |
| Priority | ❌ | Metadata (enum) |
| Completed | ❌ | Metadata (boolean) |

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

### Hızlı Başlangıç

```bash
# Clone
git clone <repo-url>
cd todo-app-yavuzlar

# Environment (örnek)
cp .env.example .env
# MASTER_ENCRYPTION_KEY, JWT_SECRET, DB credentials düzenle

# Build & Run
docker compose up --build -d

# Erişim
open http://localhost
```

### Environment Variables

| Değişken | Açıklama |
|----------|----------|
| `MASTER_ENCRYPTION_KEY` | 32-byte hex encoded AES key |
| `JWT_SECRET` | JWT imzalama anahtarı |
| `POSTGRES_*` | Database credentials |
| `BACKEND_PORT` | Backend port (default: 8080) |

---

## 📡 API Referansı

### Authentication

| Endpoint | Method | Açıklama |
|----------|--------|----------|
| `/api/v1/register` | POST | Yeni UUID üret |
| `/api/v1/login` | POST | JWT token al |

### Todos (JWT Required)

| Endpoint | Method | Açıklama |
|----------|--------|----------|
| `/api/v1/todos` | GET | Tüm todo'ları listele |
| `/api/v1/todos` | POST | Yeni todo oluştur |
| `/api/v1/todos/{id}` | PUT | Todo güncelle |
| `/api/v1/todos/{id}` | DELETE | Todo sil |

### Request/Response Format

**Create Todo:**
```json
{
  "title": "string",
  "completed": false,
  "priority": "low|medium|high",
  "due_date": "2026-01-15",
  "tags": ["work", "urgent"]
}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "decrypted title",
  "completed": false,
  "priority": "medium",
  "due_date": "2026-01-15T00:00:00Z",
  "tags": ["work", "urgent"],
  "created_at": "2026-01-11T10:00:00Z"
}
```

---

## 🎨 Frontend Yapısı

```
frontend/
├── index.html              # Ana HTML
├── styles.css              # Global stiller
├── eye-hover-animation.css # Göz animasyon stilleri
├── uuid-animation.css      # UUID reveal animasyonu
├── eye-hover-animation.js  # Progressive reveal sistemi
├── validation.js           # Frontend validation
└── js/
    ├── app.js              # Event listeners, init
    ├── auth.js             # Login/Register logic
    ├── todo.js             # CRUD operations
    ├── ui.js               # Render functions
    ├── utils.js            # Helpers
    └── notifications.js    # Toast notifications
```

### Eye Hover Animation

Göz ikonu etrafında 3 katmanlı animasyon sistemi:

| Zone | Yarıçap | Davranış |
|------|---------|----------|
| Outer | 48-87px | Yavaş rastgele karakter scramble |
| Middle | 18-51px | 2x hızda yoğun scramble |
| Inner | 0-18px | Smooth reveal - gerçek UUID görünür |

---

## 📁 Backend Yapısı

```
backend/
├── cmd/server/main.go      # Entry point
└── internal/
    ├── auth/               # JWT context
    ├── config/             # Environment config
    ├── database/           # GORM connection
    ├── encryption/
    │   ├── crypto.go       # AES-256-GCM
    │   └── aad.go          # AAD builder
    ├── handlers/           # HTTP handlers
    ├── middleware/         # Security, CORS, JWT
    ├── models/             # DB models
    ├── store/              # Repository layer
    ├── todo/               # Business logic
    └── utils/              # Logging, validation
```

---

## 📜 Lisans

MIT License

---

**Son Güncelleme**: Ocak 2026
