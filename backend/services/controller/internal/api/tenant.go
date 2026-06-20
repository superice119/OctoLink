package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/leandrofars/oktopus/internal/db"
)

// listTenants returns all tenants. Only super_admin may call this.
func (a *Api) listTenants(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != db.RoleSuperAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	tenants, err := a.db.FindAllTenants()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(tenants); err != nil {
		log.Println(err)
	}
}

// createTenant creates a new tenant. Only super_admin may call this.
func (a *Api) createTenant(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != db.RoleSuperAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var t db.Tenant
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if t.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("name is required")
		return
	}
	if t.ID == "" {
		t.ID = uuid.NewString()
	}

	if err := a.db.CreateTenant(t); err != nil {
		if err == db.ErrorTenantExists {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode("tenant already exists")
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// updateTenant updates an existing tenant. Only super_admin may call this.
func (a *Api) updateTenant(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != db.RoleSuperAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	id := mux.Vars(r)["id"]
	var t db.Tenant
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	t.ID = id

	if err := a.db.UpdateTenant(t); err != nil {
		if err == db.ErrorTenantNotFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(t)
}

// deleteTenant deletes a tenant. Only super_admin may call this.
// The default tenant cannot be deleted.
func (a *Api) deleteTenant(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(string)
	if role != db.RoleSuperAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	id := mux.Vars(r)["id"]
	if id == db.DefaultTenantID {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("cannot delete the default tenant")
		return
	}

	if err := a.db.DeleteTenant(id); err != nil {
		if err == db.ErrorTenantNotFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
