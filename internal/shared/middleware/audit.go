package middleware

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AuditMiddleware struct {
	db *sql.DB
}

func NewAuditMiddleware(db *sql.DB) *AuditMiddleware {
	return &AuditMiddleware{db: db}
}

func (m *AuditMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}
		startedAt := time.Now()
		wrapped := &auditResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next(wrapped, r)
		resourceType, resourceID := auditResource(r)
		projectID := int64(0)
		if resourceType == "project" {
			projectID = resourceID
		}
		_, err := m.db.ExecContext(r.Context(), `INSERT INTO audit_log
			(user_id, username, action, resource_type, resource_id, project_id,
			 request_method, request_path, response_code, client_ip, user_agent, duration_ms)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			GetUserID(r.Context()), GetUsername(r.Context()), auditAction(r.Method), resourceType,
			resourceID, projectID, r.Method, r.URL.Path, wrapped.status, clientIP(r),
			r.UserAgent(), time.Since(startedAt).Milliseconds())
		if err != nil {
			log.Printf("record audit log: %v", err)
		}
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *auditResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func auditAction(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

func auditResource(r *http.Request) (string, int64) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	resource := "unknown"
	if len(parts) >= 3 && parts[0] == "api" && strings.HasPrefix(parts[1], "v") {
		if parts[2] == "access" && len(parts) >= 4 {
			resource = "role_binding"
		} else {
			resource = strings.TrimSuffix(parts[2], "s")
		}
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id == 0 && resource == "service" {
		id, _ = strconv.ParseInt(r.PathValue("service_id"), 10, 64)
	}
	return resource, id
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
