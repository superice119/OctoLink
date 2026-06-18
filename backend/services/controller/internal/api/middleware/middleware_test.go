package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leandrofars/oktopus/internal/api/auth"
)

func TestMiddleware_MissingToken(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bad.token.value")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_ValidToken_InjectsContext(t *testing.T) {
	token, err := auth.GenerateJWT("user@example.com", "User", "operator", "tenant-x")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var gotEmail, gotRole, gotTenant string
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEmail, _ = r.Context().Value("email").(string)
		gotRole, _ = r.Context().Value("role").(string)
		gotTenant, _ = r.Context().Value("tenant_id").(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if gotEmail != "user@example.com" {
		t.Errorf("expected email user@example.com, got %q", gotEmail)
	}
	if gotRole != "operator" {
		t.Errorf("expected role operator, got %q", gotRole)
	}
	if gotTenant != "tenant-x" {
		t.Errorf("expected tenant_id tenant-x, got %q", gotTenant)
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	token, _ := auth.GenerateJWT("admin@example.com", "Admin", "super_admin", "default")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(RequirePermission("devices:write")(inner))

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	req.Header.Set("Authorization", token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	token, _ := auth.GenerateJWT("viewer@example.com", "Viewer", "viewer", "tenant-y")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(RequirePermission("devices:write")(inner))

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	req.Header.Set("Authorization", token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestDeviceWritePermission_ReadAlwaysAllowed(t *testing.T) {
	// viewer has devices:read but not devices:write; GET should still pass DeviceWritePermission
	token, _ := auth.GenerateJWT("v@x.com", "V", "viewer", "t1")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(DeviceWritePermission(inner))

	req := httptest.NewRequest(http.MethodGet, "/api/device", nil)
	req.Header.Set("Authorization", token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("viewer GET should be allowed by DeviceWritePermission, got %d", rr.Code)
	}
}

func TestDeviceWritePermission_WriteDeniedForViewer(t *testing.T) {
	token, _ := auth.GenerateJWT("v@x.com", "V", "viewer", "t1")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(DeviceWritePermission(inner))

	req := httptest.NewRequest(http.MethodPut, "/api/device/sn1/mqtt/set", nil)
	req.Header.Set("Authorization", token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("viewer PUT should be forbidden by DeviceWritePermission, got %d", rr.Code)
	}
}
