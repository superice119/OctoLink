package middleware

import (
	"net/http"

	"github.com/leandrofars/oktopus/internal/api/auth"
	"golang.org/x/net/context"
)

// Middleware validates the JWT and injects email, role and tenant_id into the request context.
// Non-super_admin requests with an empty tenant_id claim are rejected with 403:
// a valid token must always carry a tenant for scoped enforcement.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			info, err := auth.ValidateTokenFull(tokenString)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Non-super_admin without a tenant_id cannot be scoped correctly — reject early.
			if info.Role != "super_admin" && info.TenantID == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), "email", info.Email)
			ctx = context.WithValue(ctx, "role", info.Role)
			ctx = context.WithValue(ctx, "tenant_id", info.TenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

// RequirePermission returns a middleware that enforces a specific permission.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("role").(string)
			if !auth.CheckPermission(role, permission) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// DeviceWritePermission enforces devices:write on non-GET/HEAD requests.
func DeviceWritePermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		role, _ := r.Context().Value("role").(string)
		if !auth.CheckPermission(role, "devices:write") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
