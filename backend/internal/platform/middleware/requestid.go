package middleware

import (
	"net/http"

	"github.com/Bahjat/page-insight-tool/backend/internal/platform/requestid"
	"github.com/google/uuid"
)

// RequestID is middleware that assigns a unique request ID to each request.
// If the incoming request already carries an X-Request-ID header, that value
// is reused; otherwise a new UUID v4 is generated.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}

		ctx := requestid.NewContext(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
