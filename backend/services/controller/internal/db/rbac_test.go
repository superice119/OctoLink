package db

import (
	"testing"
)

// ---- EffectiveRole tests (no DB needed) ----

func TestEffectiveRole_ExplicitRole(t *testing.T) {
	u := User{Role: "tenant_admin", Level: NormalUser}
	if u.EffectiveRole() != "tenant_admin" {
		t.Errorf("expected tenant_admin, got %s", u.EffectiveRole())
	}
}

func TestEffectiveRole_FallbackAdminUser(t *testing.T) {
	u := User{Level: AdminUser}
	if u.EffectiveRole() != RoleSuperAdmin {
		t.Errorf("expected %s, got %s", RoleSuperAdmin, u.EffectiveRole())
	}
}

func TestEffectiveRole_FallbackNormalUser(t *testing.T) {
	u := User{Level: NormalUser}
	if u.EffectiveRole() != RoleOperator {
		t.Errorf("expected %s, got %s", RoleOperator, u.EffectiveRole())
	}
}

// ---- HasPermission tests ----

func TestHasPermission_SuperAdmin(t *testing.T) {
	perms := []string{
		PermDevicesRead, PermDevicesWrite,
		PermUsersRead, PermUsersWrite,
		PermTenantsManage, PermRolesManage,
	}
	for _, p := range perms {
		if !HasPermission(RoleSuperAdmin, p) {
			t.Errorf("super_admin should have permission %q", p)
		}
	}
}

func TestHasPermission_Viewer(t *testing.T) {
	if !HasPermission(RoleViewer, PermDevicesRead) {
		t.Error("viewer should have devices:read")
	}
	if HasPermission(RoleViewer, PermDevicesWrite) {
		t.Error("viewer should NOT have devices:write")
	}
}

func TestHasPermission_Unknown(t *testing.T) {
	if HasPermission("nobody", PermDevicesRead) {
		t.Error("unknown role should not have any permission")
	}
}

func TestBuiltinRolePermissions_AllRolesPresent(t *testing.T) {
	roles := []string{RoleSuperAdmin, RoleTenantAdmin, RoleOperator, RoleViewer}
	for _, r := range roles {
		if _, ok := BuiltinRolePermissions[r]; !ok {
			t.Errorf("built-in role %q missing from BuiltinRolePermissions", r)
		}
	}
}

// ---- EffectiveTenantID tests ----

func TestEffectiveTenantID_WithExplicitTenant(t *testing.T) {
	u := User{TenantID: "acme", Role: RoleOperator}
	if u.EffectiveTenantID() != "acme" {
		t.Errorf("expected acme, got %q", u.EffectiveTenantID())
	}
}

func TestEffectiveTenantID_EmptyNonSuperAdmin_FallsBackToDefault(t *testing.T) {
	u := User{TenantID: "", Role: RoleOperator}
	if u.EffectiveTenantID() != DefaultTenantID {
		t.Errorf("non-super_admin with empty TenantID should fall back to %q, got %q", DefaultTenantID, u.EffectiveTenantID())
	}
}

func TestEffectiveTenantID_EmptyTenantAdmin_FallsBackToDefault(t *testing.T) {
	u := User{TenantID: "", Role: RoleTenantAdmin}
	if u.EffectiveTenantID() != DefaultTenantID {
		t.Errorf("tenant_admin with empty TenantID should fall back to %q, got %q", DefaultTenantID, u.EffectiveTenantID())
	}
}

func TestEffectiveTenantID_SuperAdmin_EmptyPreserved(t *testing.T) {
	u := User{TenantID: "", Role: RoleSuperAdmin}
	// super_admin is cross-tenant; empty tenant_id is valid
	if u.EffectiveTenantID() != "" {
		t.Errorf("super_admin with empty TenantID should return empty string, got %q", u.EffectiveTenantID())
	}
}

func TestEffectiveTenantID_LegacyAdminUser_EmptyPreserved(t *testing.T) {
	u := User{Level: AdminUser} // no Role set → EffectiveRole() == super_admin
	if u.EffectiveTenantID() != "" {
		t.Errorf("legacy AdminUser with empty TenantID should return empty string, got %q", u.EffectiveTenantID())
	}
}

// ---- IsGlobalRole tests ----

func TestIsGlobalRole_SuperAdminIsGlobal(t *testing.T) {
	if !IsGlobalRole(RoleSuperAdmin) {
		t.Error("super_admin must be a global role")
	}
}

func TestIsGlobalRole_TenantRolesAreNotGlobal(t *testing.T) {
	for _, r := range []string{RoleTenantAdmin, RoleOperator, RoleViewer} {
		if IsGlobalRole(r) {
			t.Errorf("role %q must NOT be a global role", r)
		}
	}
}
