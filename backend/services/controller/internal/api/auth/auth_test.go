package auth

import (
	"os"
	"testing"
)

// TestMain sets a valid signing secret so token-related tests do not hit the
// fail-closed guard in getJwtKey (WS-38).
func TestMain(m *testing.M) {
	os.Setenv("SECRET_API_KEY", "test-secret-key-for-unit-tests-only")
	os.Exit(m.Run())
}

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

func TestRequireSecret(t *testing.T) {
	// Restore the package-wide test secret afterwards so the fail-closed
	// getJwtKey guard does not trip later tests.
	orig, had := os.LookupEnv("SECRET_API_KEY")
	t.Cleanup(func() {
		if had {
			os.Setenv("SECRET_API_KEY", orig)
		} else {
			os.Unsetenv("SECRET_API_KEY")
		}
	})

	cases := []struct {
		name    string
		value   string
		unset   bool
		wantErr bool
	}{
		{name: "unset", unset: true, wantErr: true},
		{name: "empty", value: "", wantErr: true},
		{name: "insecure default", value: "supersecretkey", wantErr: true},
		{name: "too short", value: "short", wantErr: true},
		{name: "valid", value: "a-sufficiently-long-random-secret", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unset {
				os.Unsetenv("SECRET_API_KEY")
			} else {
				os.Setenv("SECRET_API_KEY", tc.value)
			}
			err := RequireSecret()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tc.name, err)
			}
		})
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	_, err := ValidateToken("not.a.valid.jwt")
	if err == nil {
		t.Error("expected error for invalid JWT, got nil")
	}
}
