package middleware

import (
	"net/http"
	"strings"

	"todo-app-backend/internal/utils"
)

var contentTypeLogger = utils.NewLogger("CONTENT_TYPE")

// ContentTypeMiddleware validates Content-Type header based on endpoint and HTTP method.
// Currently the backend exposes:
//   - POST /api/v2/register  (JSON body, but Phase 1 allows empty body without Content-Type)
//   - POST /api/v2/login     (JSON body, Content-Type: application/json)
//   - GET  /api/v2/todos     (no body)
//   - POST /api/v2/todos     (JSON body, Content-Type: application/json)
//   - PUT  /api/v2/todos/:id (JSON body, Content-Type: application/json)
//   - DELETE /api/v2/todos/:id (no body)
//
// For diğer path/method kombinasyonları şu an whitelist uygulanmaz; böylece sistemi
// bozmadan gelecekteki endpoint eklemeleri için esnek kalır.
func ContentTypeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.Path

		// Sadece POST ve PUT için Content-Type kontrolü yapıyoruz
		if method != http.MethodPost && method != http.MethodPut {
			next(w, r)
			return
		}

		contentTypeLogger.Debug("ContentTypeMiddleware: method=%s path=%s content-length=%d", method, path, r.ContentLength)

		// Endpoint bazlı Content-Type ve body gereksinimleri
		requireJSON := false
		allowEmptyBodyWithoutContentType := false

		switch {
		// POST /api/v2/register
		// Phase 1'de body tamamen boş olabiliyor (AccountNumber preview),
		// bu durumda Content-Type header'ı zorunlu değil.
		case path == "/api/v2/register" && method == http.MethodPost:
			requireJSON = true
			if r.ContentLength == 0 {
				allowEmptyBodyWithoutContentType = true
			}

		// POST /api/v2/login (her zaman JSON body bekleniyor)
		case path == "/api/v2/login" && method == http.MethodPost:
			requireJSON = true

		// POST /api/v2/todos (JSON body)
		case path == "/api/v2/todos" && method == http.MethodPost:
			requireJSON = true

		// PUT /api/v2/todos/:id (JSON body)
		case strings.HasPrefix(path, "/api/v2/todos/") && method == http.MethodPut:
			requireJSON = true

		// Diğer path/method kombinasyonları için şu an Content-Type whitelist uygulanmıyor.
		default:
			contentTypeLogger.Debug("ContentTypeMiddleware: skipping Content-Type check for method=%s path=%s", method, path)
			next(w, r)
			return
		}

		// Eğer bu endpoint için JSON bekleniyor ama body gerçekten boşsa
		// ve boş body kabul edilen bir case ise, Content-Type zorunlu değil.
		if allowEmptyBodyWithoutContentType && r.ContentLength == 0 {
			contentTypeLogger.Debug("ContentTypeMiddleware: empty body allowed without Content-Type for method=%s path=%s", method, path)
			next(w, r)
			return
		}

		if requireJSON {
			contentType := r.Header.Get("Content-Type")
			// Remove charset and other parameters (e.g., "application/json; charset=utf-8" -> "application/json")
			baseContentType := strings.TrimSpace(strings.Split(contentType, ";")[0])

			// Whitelist: Only allow "application/json"
			const allowedContentType = "application/json"
			if baseContentType != allowedContentType {
				contentTypeLogger.Warn("ContentTypeMiddleware: unsupported content-type=%s (base=%s) for method=%s path=%s", contentType, baseContentType, method, path)
				http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
				return
			}
		}

		next(w, r)
	}
}
