package server

import (
	"crypto/hmac"
	"net/http"
	"strings"
)

// requireIngestionKey rejects any request without a valid
// "Authorization: Bearer <Ingestion key>" header.
func (s *Server) requireIngestionKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || key == "" || s.ingestionKey == "" || !hmac.Equal([]byte(key), []byte(s.ingestionKey)) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
