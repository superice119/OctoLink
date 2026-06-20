package auth

import (
	"testing"
)

func TestCheckPermission_SuperAdmin(t *testing.T) {
	allPerms := []string{
		"devices:read", "devices:write",
		"users:read", "users:write",
		"tenants:manage", "roles:manage",
	}
	for _, p := range allPerms {
		if !CheckPermission("super_admin", p) {
			t.Errorf("super_admin should have permission %q", p)
		}
	}
}

func TestCheckPermission_TenantAdmin(t *testing.T) {
	allowed := []string{"devices:read", "devices:write", "users:read", "users:write"}
	denied := []string{"tenants:manage", "roles:manage"}

	for _, p := range allowed {
		if !CheckPermission("tenant_admin", p) {
			t.Errorf("tenant_admin should have permission %q", p)
		}
	}
	for _, p := range denied {
		if CheckPermission("tenant_admin", p) {
			t.Errorf("tenant_admin should NOT have permission %q", p)
		}
	}
}

func TestCheckPermission_Operator(t *testing.T) {
	allowed := []string{"devices:read", "devices:write"}
	denied := []string{"users:read", "users:write", "tenants:manage", "roles:manage"}

	for _, p := range allowed {
		if !CheckPermission("operator", p) {
			t.Errorf("operator should have permission %q", p)
		}
	}
	for _, p := range denied {
		if CheckPermission("operator", p) {
			t.Errorf("operator should NOT have permission %q", p)
		}
	}
}

func TestCheckPermission_Viewer(t *testing.T) {
	if !CheckPermission("viewer", "devices:read") {
		t.Error("viewer should have devices:read")
	}
	denied := []string{"devices:write", "users:read", "users:write", "tenants:manage", "roles:manage"}
	for _, p := range denied {
		if CheckPermission("viewer", p) {
			t.Errorf("viewer should NOT have permission %q", p)
		}
	}
}

func TestCheckPermission_UnknownRole(t *testing.T) {
	if CheckPermission("hacker", "devices:read") {
		t.Error("unknown role should not have any permission")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	token, err := GenerateJWT("test@example.com", "Test User", "operator", "tenant-abc")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	info, err := ValidateTokenFull(token)
	if err != nil {
		t.Fatalf("ValidateTokenFull failed: %v", err)
	}

	if info.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", info.Email)
	}
	if info.Role != "operator" {
		t.Errorf("expected role operator, got %s", info.Role)
	}
	if info.TenantID != "tenant-abc" {
		t.Errorf("expected tenant_id tenant-abc, got %s", info.TenantID)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	_, err := ValidateToken("not.a.valid.jwt")
	if err == nil {
		t.Error("expected error for invalid JWT, got nil")
	}
}
