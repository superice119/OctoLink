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
