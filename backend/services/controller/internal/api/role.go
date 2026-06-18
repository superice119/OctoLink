package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/leandrofars/oktopus/internal/db"
)

// listRoles returns all roles visible to the caller.
// super_admin sees all; tenant_admin sees system roles + their tenant's custom roles.
func (a *Api) listRoles(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	tenantID, _ := r.Context().Value("tenant_id").(string)

	queryTenantID := ""
	if role != db.RoleSuperAdmin {
		queryTenantID = tenantID
	}

	roles, err := a.db.FindAllRoles(queryTenantID)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(roles); err != nil {
		log.Println(err)
	}
}

// createRole creates a custom role. System roles cannot be created via API.
// Only super_admin or tenant_admin may create roles.
func (a *Api) createRole(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value("role").(string)
	callerTenantID, _ := r.Context().Value("tenant_id").(string)

	if callerRole != db.RoleSuperAdmin && callerRole != db.RoleTenantAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var newRole db.Role
	if err := json.NewDecoder(r.Body).Decode(&newRole); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if newRole.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("name is required")
		return
	}
	if newRole.ID == "" {
		newRole.ID = uuid.NewString()
	}
	newRole.IsSystem = false

	// tenant_admin can only create roles for their own tenant
	if callerRole == db.RoleTenantAdmin {
		newRole.TenantID = callerTenantID
	}

	if err := a.db.CreateRole(newRole); err != nil {
		if err == db.ErrorRoleExists {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode("role already exists")
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newRole)
}

// updateRole updates a custom role's permissions. System roles cannot be modified.
func (a *Api) updateRole(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value("role").(string)
	callerTenantID, _ := r.Context().Value("tenant_id").(string)

	if callerRole != db.RoleSuperAdmin && callerRole != db.RoleTenantAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	id := mux.Vars(r)["id"]
	existing, err := a.db.FindRole(id)
	if err == db.ErrorRoleNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if existing.IsSystem {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode("system roles cannot be modified")
		return
	}

	// tenant_admin can only update roles in their tenant
	if callerRole == db.RoleTenantAdmin && existing.TenantID != callerTenantID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var updated db.Role
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	updated.ID = id

	if err := a.db.UpdateRole(updated); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

// deleteRole deletes a custom role. System roles cannot be deleted.
func (a *Api) deleteRole(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value("role").(string)
	callerTenantID, _ := r.Context().Value("tenant_id").(string)

	if callerRole != db.RoleSuperAdmin && callerRole != db.RoleTenantAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	id := mux.Vars(r)["id"]
	existing, err := a.db.FindRole(id)
	if err == db.ErrorRoleNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if existing.IsSystem {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode("system roles cannot be deleted")
		return
	}

	// tenant_admin can only delete roles in their tenant
	if callerRole == db.RoleTenantAdmin && existing.TenantID != callerTenantID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := a.db.DeleteRole(id); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// assignUserRole updates a user's role and tenant. Only super_admin or tenant_admin may call this.
func (a *Api) assignUserRole(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value("role").(string)
	callerTenantID, _ := r.Context().Value("tenant_id").(string)

	if callerRole != db.RoleSuperAdmin && callerRole != db.RoleTenantAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	type assignRoleReq struct {
		Email    string `json:"email"`
		Role     string `json:"role"`
		TenantID string `json:"tenant_id"`
	}

	var req assignRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// tenant_admin can only assign users within their own tenant
	targetTenant := req.TenantID
	if callerRole == db.RoleTenantAdmin {
		targetTenant = callerTenantID
	}

	if err := a.db.UpdateUserRole(req.Email, req.Role, targetTenant); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
