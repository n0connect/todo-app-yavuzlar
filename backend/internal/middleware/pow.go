package middleware

import (
	"encoding/json"
	"net/http"

	"todo-app-backend/internal/models"
	"todo-app-backend/internal/pow"
	"todo-app-backend/internal/utils"
)

var powMiddlewareLogger = utils.NewLogger("POW_MIDDLEWARE")

// PoWMiddleware validates Proof of Work solution for endpoints that require it
func PoWMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		powMiddlewareLogger.Debug("PoWMiddleware: checking PoW for path: %s", r.URL.Path)

		// For registration endpoint, check if PoW is required
		if r.URL.Path == "/api/v2/register" && r.Method == http.MethodPost {
			// Temporarily parse request body to check if PoW is provided
			// We'll let the actual handler parse it again to avoid body consumption issues
			bodyBytes, err := utils.ReadRequestBody(r)
			if err != nil {
				powMiddlewareLogger.LogError("Reading request body", err)
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			// Parse the body to check for PoW
			var req models.RegisterRequest
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				// If JSON is invalid but it's phase 1, we still need to check
				// For now, we'll let the handler deal with invalid JSON
				powMiddlewareLogger.Debug("PoWMiddleware: could not parse request body, letting handler handle it")
			} else {
				// If this is phase 1 registration (generate account number) and PoW is provided, validate it
				if !req.Confirm && req.PoW != nil {
					powMiddlewareLogger.Debug("PoWMiddleware: validating PoW solution for registration phase 1")

					// Validate the PoW solution
					if !pow.ValidateSolution(req.PoW) {
						powMiddlewareLogger.Warn("PoWMiddleware: invalid PoW solution for registration")
						http.Error(w, "Invalid proof of work", http.StatusPreconditionFailed)
						return
					}

					powMiddlewareLogger.Debug("PoWMiddleware: PoW solution validated successfully")
				}
			}

			// Restore the body for the next handler
			r.Body = utils.MakeReadCloser(bodyBytes)
		}

		// For other endpoints that might require PoW in the future, add checks here

		next(w, r)
	}
}