package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

// SyncPathValues exposes go-zero route parameters through the Go 1.22
// Request.PathValue API used by AxiOps handlers and audit resource parsing.
func SyncPathValues(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for name, value := range pathvar.Vars(r) {
			r.SetPathValue(name, value)
		}
		next(w, r)
	}
}
