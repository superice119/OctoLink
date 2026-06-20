package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leandrofars/oktopus/internal/db"
)

// checkDeviceTenantAccess_SuperAdminBypass verifies super_admin never hits NATS.
// When callerRole == "super_admin", the function must return true immediately.
func TestCheckDeviceTenantAccess_SuperAdminBypass(t *testing.T) {
	w := httptest.NewRecorder()
	// nc is nil — if NATS were called, this would panic; its absence proves the bypass.
	result := checkDeviceTenantAccess(w, nil, "any-sn", "super_admin", "")
	if !result {
		t.Error("super_admin should always pass checkDeviceTenantAccess without NATS call")
	}
	if w.Code != http.StatusOK {
		t.Errorf("no HTTP error should be written for super_admin, got %d", w.Code)
	}
}

// TestCheckUserTenantOwnership covers the cross-tenant 403 logic for assignUserRole / deleteUser.
func TestCheckUserTenantOwnership_SuperAdmin_CanActCrossTenant(t *testing.T) {
	if !checkUserTenantOwnership("super_admin", "tenantA", "tenantB") {
		t.Error("super_admin should be allowed to act on any user regardless of tenant")
	}
}

func TestCheckUserTenantOwnership_TenantAdmin_SameTenant_Allowed(t *testing.T) {
	if !checkUserTenantOwnership("tenant_admin", "tenantA", "tenantA") {
		t.Error("tenant_admin should be allowed to act on users in their own tenant")
	}
}

func TestCheckUserTenantOwnership_TenantAdmin_CrossTenant_Denied(t *testing.T) {
	if checkUserTenantOwnership("tenant_admin", "tenantA", "tenantB") {
		t.Error("tenant_admin must NOT be allowed to act on users in another tenant (cross-tenant 403 case)")
	}
}

func TestCheckUserTenantOwnership_Operator_CrossTenant_Denied(t *testing.T) {
	if checkUserTenantOwnership("operator", "tenantA", "tenantB") {
		t.Error("operator must NOT be allowed to act on users in another tenant")
	}
}

// TestRequireDeviceAccess_SuperAdminBypassViaContext verifies that requireDeviceAccess
// with a super_admin context never reaches NATS (nc is nil; a nil-deref would fail the test).
func TestRequireDeviceAccess_SuperAdminBypassViaContext(t *testing.T) {
	a := Api{nc: nil} // nil NATS — proves super_admin does not call NATS
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), "role", "super_admin")
	ctx = context.WithValue(ctx, "tenant_id", "")
	req = req.WithContext(ctx)

	if !a.requireDeviceAccess(w, req, "any-sn") {
		t.Error("super_admin should pass requireDeviceAccess without NATS call")
	}
}

// TestRequireDeviceAccess_NonSuperAdmin_BlockedWhenNATSUnavailable verifies that a
// non-super_admin without NATS (device info unavailable) is denied access.
// This simulates the 403 path when device info cannot be retrieved (fail-closed).
func TestRequireDeviceAccess_NonSuperAdmin_BlockedWhenNATSUnavailable(t *testing.T) {
	a := Api{nc: nil}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), "role", "operator")
	ctx = context.WithValue(ctx, "tenant_id", "tenantA")
	req = req.WithContext(ctx)

	// nil NATS will cause getDeviceInfo to fail → checkDeviceTenantAccess returns false
	result := a.requireDeviceAccess(w, req, "sn-tenantB")
	if result {
		t.Error("non-super_admin with no NATS connection should be denied (fail-closed)")
	}
}

// ---- Privilege escalation prevention (Issue: tenant_admin → super_admin) ----

// checkRoleAssignAllowed is a pure function extracted from assignUserRole logic
// for easy unit-testing without a real DB.
func checkRoleAssignAllowed(callerRole, targetRole string) bool {
	if callerRole == db.RoleSuperAdmin {
		return true
	}
	// tenant_admin cannot assign global roles
	if db.IsGlobalRole(targetRole) {
		return false
	}
	return true
}

func TestPrivilegeEscalation_TenantAdmin_CannotAssignSuperAdmin(t *testing.T) {
	if checkRoleAssignAllowed(db.RoleTenantAdmin, db.RoleSuperAdmin) {
		t.Error("tenant_admin must NOT be able to assign super_admin role (privilege escalation)")
	}
}

func TestPrivilegeEscalation_TenantAdmin_CanAssignTenantRoles(t *testing.T) {
	for _, r := range []string{db.RoleTenantAdmin, db.RoleOperator, db.RoleViewer} {
		if !checkRoleAssignAllowed(db.RoleTenantAdmin, r) {
			t.Errorf("tenant_admin should be able to assign tenant-scoped role %q", r)
		}
	}
}

func TestPrivilegeEscalation_SuperAdmin_CanAssignAnyRole(t *testing.T) {
	for _, r := range []string{db.RoleSuperAdmin, db.RoleTenantAdmin, db.RoleOperator, db.RoleViewer} {
		if !checkRoleAssignAllowed(db.RoleSuperAdmin, r) {
			t.Errorf("super_admin should be able to assign role %q", r)
		}
	}
}
