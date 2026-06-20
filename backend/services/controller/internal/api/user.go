package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/mail"

	"github.com/gorilla/mux"
	"github.com/leandrofars/oktopus/internal/api/auth"
	"github.com/leandrofars/oktopus/internal/db"
	"github.com/leandrofars/oktopus/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (a *Api) retrieveUsers(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value("role").(string)
	callerTenantID, _ := r.Context().Value("tenant_id").(string)

	var (
		users []map[string]interface{}
		err   error
	)

	// super_admin sees all users; everyone else only sees their tenant
	if callerRole == db.RoleSuperAdmin {
		users, err = a.db.FindAllUsers()
	} else {
		users, err = a.db.FindUsersByTenant(callerTenantID)
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, x := range users {
		objectID, ok := x["_id"].(primitive.ObjectID)
		if ok {
			creationTime := objectID.Timestamp()
			x["createdAt"] = creationTime.Format("02/01/2006")
		}
		delete(x, "password")
	}

	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		log.Println(err)
	}
}

func (a *Api) registerUser(w http.ResponseWriter, r *http.Request) {

	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	email, err := auth.ValidateToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Check if user which is requesting creation has the necessary privileges
	rUser, err := a.db.FindUser(email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	callerRole := rUser.EffectiveRole()
	if callerRole != db.RoleSuperAdmin && callerRole != db.RoleTenantAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var user db.User
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Assign defaults: new users created by tenant_admin belong to that tenant
	if user.Role == "" {
		user.Role = db.RoleOperator
	}
	if user.TenantID == "" {
		user.TenantID = rUser.TenantID
		if user.TenantID == "" {
			user.TenantID = db.DefaultTenantID
		}
	}
	// tenant_admin cannot create users outside their own tenant and cannot
	// grant global roles (privilege escalation prevention)
	if callerRole == db.RoleTenantAdmin {
		user.TenantID = rUser.TenantID
		if db.IsGlobalRole(user.Role) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode("tenant_admin cannot create users with global roles such as super_admin")
			return
		}
	}
	// Keep legacy Level field in sync
	user.Level = db.NormalUser

	if err := user.HashPassword(user.Password); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if user.Email == "" || user.Password == "" || !valid(user.Email) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := a.db.RegisterUser(user); err != nil {
		if err == db.ErrorUserExists {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("User with this email already exists"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func valid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (a *Api) deleteUser(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	email, err := auth.ValidateToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rUser, err := a.db.FindUser(email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	callerRole := rUser.EffectiveRole()
	userEmail := mux.Vars(r)["user"]

	// Users can always delete themselves
	if rUser.Email == userEmail {
		if err := a.db.DeleteUser(userEmail); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
		}
		return
	}

	if callerRole == db.RoleSuperAdmin {
		if err := a.db.DeleteUser(userEmail); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
		}
		return
	}

	if callerRole == db.RoleTenantAdmin {
		// Verify target user belongs to the same tenant before deleting
		targetUser, err := a.db.FindUser(userEmail)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if !checkUserTenantOwnership(callerRole, rUser.TenantID, targetUser.TenantID) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if err := a.db.DeleteUser(userEmail); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
		}
		return
	}

	w.WriteHeader(http.StatusForbidden)
}

func (a *Api) changePassword(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	email, err := auth.ValidateToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var user db.User
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder(err, w)
		return
	}
	user.Email = email

	if len(user.Password) < 8 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Password must be at least 8 characters long"))
		return
	}

	if err := user.HashPassword(user.Password); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := a.db.UpdatePassword(user); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) registerAdminUser(w http.ResponseWriter, r *http.Request) {

	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		users, err := a.db.FindAllUsers()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			utils.MarshallEncoder(err, w)
		}

		if !adminUserExists(users) {
			var user db.User
			err = json.NewDecoder(r.Body).Decode(&user)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			user.Level = db.AdminUser
			user.Role = db.RoleSuperAdmin
			user.TenantID = db.DefaultTenantID

			if err := user.HashPassword(user.Password); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if err := a.db.RegisterUser(user); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(http.StatusForbidden)
		}

		return
	}

	email, err := auth.ValidateToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//Check if user which is requesting creation has the necessary privileges
	rUser, err := a.db.FindUser(email)
	if rUser.EffectiveRole() != db.RoleSuperAdmin {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var user db.User
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user.Level = db.AdminUser
	user.Role = db.RoleSuperAdmin
	if user.TenantID == "" {
		user.TenantID = db.DefaultTenantID
	}

	if err := user.HashPassword(user.Password); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := a.db.RegisterUser(user); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func adminUserExists(users []map[string]interface{}) bool {

	if len(users) == 0 {
		return false
	}

	for _, x := range users {
		if db.UserLevels(x["level"].(int32)) == db.AdminUser {
			return true
		}
	}
	return false
}

func (a *Api) adminUserExists(w http.ResponseWriter, r *http.Request) {

	users, err := a.db.FindAllUsers()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	adminExits := adminUserExists(users)
	json.NewEncoder(w).Encode(adminExits)
	return
}

type TokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *Api) generateToken(w http.ResponseWriter, r *http.Request) {
	var tokenReq TokenRequest

	err := json.NewDecoder(r.Body).Decode(&tokenReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := a.db.FindUser(tokenReq.Email)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("Invalid Credentials")
		return
	}

	credentialError := user.CheckPassword(tokenReq.Password)
	if credentialError != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("Invalid Credentials")
		return
	}

	token, err := auth.GenerateJWT(user.Email, user.Name, user.EffectiveRole(), user.EffectiveTenantID())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
	return
}
