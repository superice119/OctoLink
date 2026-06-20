package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/leandrofars/oktopus/internal/api/cors"
	"github.com/leandrofars/oktopus/internal/api/middleware"
	"github.com/leandrofars/oktopus/internal/bridge"
	"github.com/leandrofars/oktopus/internal/config"
	"github.com/leandrofars/oktopus/internal/db"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Api struct {
	port    string
	js      jetstream.JetStream
	nc      *nats.Conn
	bridge  bridge.Bridge
	db      db.Database
	kv      jetstream.KeyValue
	paramKv jetstream.KeyValue
	ctx     context.Context
}

const REQUEST_TIMEOUT = time.Second * 30

func NewApi(c *config.Config, js jetstream.JetStream, nc *nats.Conn, bridge bridge.Bridge, d db.Database, kv jetstream.KeyValue) Api {
	paramKv, err := js.CreateOrUpdateKeyValue(c.RestApi.Ctx, jetstream.KeyValueConfig{
		Bucket:      "usp-param-cache",
		Description: "USP parameter GET result cache (5-min TTL)",
		TTL:         5 * time.Minute,
	})
	if err != nil {
		log.Printf("Warning: failed to create USP param cache KV bucket, caching disabled: %v", err)
		paramKv = nil
	}

	return Api{
		port:    c.RestApi.Port,
		js:      js,
		nc:      nc,
		ctx:     c.RestApi.Ctx,
		bridge:  bridge,
		db:      d,
		kv:      kv,
		paramKv: paramKv,
	}
}

func (a *Api) StartApi() {
	r := mux.NewRouter()
	authentication := r.PathPrefix("/api/auth").Subrouter()
	authentication.HandleFunc("/login", a.generateToken).Methods("PUT")
	authentication.HandleFunc("/register", a.registerUser).Methods("POST")
	authentication.HandleFunc("/delete/{user}", a.deleteUser).Methods("DELETE")
	authentication.HandleFunc("/password/{user}", a.changePassword).Methods("PUT")
	authentication.HandleFunc("/password", a.changePassword).Methods("PUT")
	authentication.HandleFunc("/admin/register", a.registerAdminUser).Methods("POST")
	authentication.HandleFunc("/admin/exists", a.adminUserExists).Methods("GET")

	iot := r.PathPrefix("/api/device").Subrouter()
	iot.HandleFunc("/alias", a.setDeviceAlias).Methods("PUT")
	iot.HandleFunc("/auth", a.deviceAuth).Methods("GET", "POST", "DELETE")
	iot.HandleFunc("/message/{type}", a.addTemplate).Methods("POST")
	iot.HandleFunc("/message", a.updateTemplate).Methods("PUT")
	iot.HandleFunc("/message", a.getTemplate).Methods("GET")
	iot.HandleFunc("/message", a.deleteTemplate).Methods("DELETE")
	iot.HandleFunc("/cwmp/{sn}/generic", a.cwmpGenericMsg).Methods("PUT")
	iot.HandleFunc("/cwmp/{sn}/getParameterNames", a.cwmpGetParameterNamesMsg).Methods("PUT")
	iot.HandleFunc("/cwmp/{sn}/getParameterValues", a.cwmpGetParameterValuesMsg).Methods("PUT")
	iot.HandleFunc("/cwmp/{sn}/getParameterAttributes", a.cwmpGetParameterAttributesMsg).Methods("PUT")
	iot.HandleFunc("/cwmp/{sn}/setParameterValues", a.cwmpSetParameterValuesMsg).Methods("PUT")
	iot.HandleFunc("/cwmp/{sn}/addObject", a.cwmpAddObjectMsg).Methods("PUT")
	iot.HandleFunc("/cwmp/{sn}/deleteObject", a.cwmpDeleteObjectMsg).Methods("PUT")
	iot.HandleFunc("", a.retrieveDevices).Methods("GET", "DELETE")
	iot.HandleFunc("/filterOptions", a.filterOptions).Methods("GET")
	iot.HandleFunc("/{sn}/{mtp}/generic", a.deviceGenericMessage).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/get/cached", a.deviceGetCachedMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/get", a.deviceGetMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/add", a.deviceCreateMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/del", a.deviceDeleteMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/set", a.deviceUpdateMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/notify", a.deviceNotifyMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/parameters", a.deviceGetSupportedParametersMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/instances", a.deviceGetParameterInstances).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/operate", a.deviceOperateMsg).Methods("PUT")
	iot.HandleFunc("/{sn}/{mtp}/fw_update", a.deviceFwUpdate).Methods("PUT")
	iot.HandleFunc("/{sn}/wifi", a.deviceWifi).Methods("PUT", "GET")

	dash := r.PathPrefix("/api/info").Subrouter()
	dash.HandleFunc("/vendors", a.vendorsInfo).Methods("GET")
	dash.HandleFunc("/status", a.statusInfo).Methods("GET")
	dash.HandleFunc("/device_class", a.productClassInfo).Methods("GET")
	dash.HandleFunc("/general", a.generalInfo).Methods("GET")

	users := r.PathPrefix("/api/users").Subrouter()
	users.HandleFunc("", a.retrieveUsers).Methods("GET")

	// RBAC: tenant management (super_admin only — enforced inside handlers)
	tenants := r.PathPrefix("/api/tenants").Subrouter()
	tenants.HandleFunc("", a.listTenants).Methods("GET")
	tenants.HandleFunc("", a.createTenant).Methods("POST")
	tenants.HandleFunc("/{id}", a.updateTenant).Methods("PUT")
	tenants.HandleFunc("/{id}", a.deleteTenant).Methods("DELETE")

	// RBAC: role management
	roles := r.PathPrefix("/api/roles").Subrouter()
	roles.HandleFunc("", a.listRoles).Methods("GET")
	roles.HandleFunc("", a.createRole).Methods("POST")
	roles.HandleFunc("/{id}", a.updateRole).Methods("PUT")
	roles.HandleFunc("/{id}", a.deleteRole).Methods("DELETE")
	roles.HandleFunc("/assign", a.assignUserRole).Methods("POST")

	/* ----- Middleware for requests which requires user to be authenticated ---- */
	authMiddleware := func(handler http.Handler) http.Handler {
		return middleware.Middleware(handler)
	}

	iot.Use(authMiddleware)
	// Enforce devices:read on all device routes; devices:write on write methods
	iot.Use(middleware.RequirePermission("devices:read"))
	iot.Use(middleware.DeviceWritePermission)

	dash.Use(authMiddleware)
	dash.Use(middleware.RequirePermission("devices:read"))

	users.Use(authMiddleware)
	users.Use(middleware.RequirePermission("users:read"))

	tenants.Use(authMiddleware)
	// Fine-grained checks are done inside handler (requires tenants:manage / super_admin)

	roles.Use(authMiddleware)
	// Fine-grained checks are done inside handler (requires roles:manage for write ops)
	/* -------------------------------------------------------------------------- */

	corsOpts := cors.GetCorsConfig()

	srv := &http.Server{
		Addr:         "0.0.0.0:" + a.port,
		WriteTimeout: time.Second * 60,
		ReadTimeout:  time.Second * 60,
		IdleTimeout:  time.Second * 60,
		Handler:      corsOpts.Handler(r),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	log.Println("Running REST API at port", a.port)
}
