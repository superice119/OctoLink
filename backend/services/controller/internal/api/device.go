package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/leandrofars/oktopus/internal/bridge"
	"github.com/leandrofars/oktopus/internal/db"
	"github.com/leandrofars/oktopus/internal/entity"
	local "github.com/leandrofars/oktopus/internal/nats"
	"github.com/leandrofars/oktopus/internal/utils"
	"github.com/nats-io/nats.go/jetstream"
	"go.mongodb.org/mongo-driver/bson"
)

func (a *Api) retrieveDevices(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value("role").(string)
	callerTenantID, _ := r.Context().Value("tenant_id").(string)

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			err := json.NewEncoder(w).Encode("No id provided")
			if err != nil {
				log.Println(err)
			}
			return
		}

		ids := strings.Split(id, ",")

		// For non-super_admin, verify each device belongs to the caller's tenant
		if callerRole != db.RoleSuperAdmin {
			for _, sn := range ids {
				device, err := getDeviceInfo(w, sn, a.nc)
				if err != nil {
					return
				}
				if device.Customer != callerTenantID {
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode("device " + sn + " does not belong to your tenant")
					return
				}
			}
		}

		msg, err := bridge.NatsReq[int64](local.NATS_ADAPTER_SUBJECT+"devices.delete", utils.Marshall(ids), w, a.nc)
		if err != nil {
			return
		}

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"number_of_deleted_devices": msg.Msg,
		})
		if err != nil {
			log.Println(err)
		}
		return
	}

	const PAGE_SIZE_LIMIT = 50
	const PAGE_SIZE_DEFAULT = 20

	// Get specific device
	id := r.URL.Query().Get("id")
	if id != "" {
		device, err := getDeviceInfo(w, id, a.nc)
		if err != nil {
			return
		}
		// Tenant isolation: non-super_admin can only view devices in their tenant
		if callerRole != db.RoleSuperAdmin && device.Customer != callerTenantID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode("device does not belong to your tenant")
			return
		}
		err = json.NewEncoder(w).Encode(device)
		if err != nil {
			log.Println(err)
		}
		return
	}

	statusOrderFromUser := r.URL.Query().Get("statusOrder")
	var statusOrder int
	if statusOrderFromUser != "" {
		if statusOrderFromUser == "asc" {
			statusOrder = 1
		} else if statusOrderFromUser == "desc" {
			statusOrder = -1
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("Status order must be 'asc' or 'desc'")
			return
		}
	} else {
		statusOrder = 1
	}

	sort := bson.M{}
	sort["status"] = statusOrder

	version := r.URL.Query().Get("version")
	vendor := r.URL.Query().Get("vendor")
	productClass := r.URL.Query().Get("type")
	alias := r.URL.Query().Get("alias")
	model := r.URL.Query().Get("model")
	status := r.URL.Query().Get("status")

	// Get devices with pagination
	page_n := r.URL.Query().Get("page_number")
	page_s := r.URL.Query().Get("page_size")

	var err error

	var page_number int64
	if page_n == "" {
		page_number = 0
	} else {
		page_number, err = strconv.ParseInt(page_n, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("Page number must be an integer")
			return
		}
	}

	var page_size int64
	if page_s != "" {
		page_size, err = strconv.ParseInt(page_s, 10, 64)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("Page size must be an integer")
			return
		}

		if page_size > PAGE_SIZE_LIMIT {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("Page size must not exceed " + strconv.Itoa(PAGE_SIZE_LIMIT))
			return
		}

	} else {
		page_size = PAGE_SIZE_DEFAULT
	}

	skip := page_number * (page_size - 1)

	filter := map[string]interface{}{
		"status_order": statusOrder,
		"limit":        page_size,
		"skip":         skip,
	}

	// Tenant isolation: non-super_admin only see devices in their tenant.
	// After middleware enforcement, callerTenantID is always non-empty for non-super_admin.
	if callerRole != db.RoleSuperAdmin {
		filter["customer"] = callerTenantID
	}

	if version != "" {
		filter["version"] = version
	}
	if vendor != "" {
		filter["vendor"] = vendor
	}
	if productClass != "" {
		filter["productClass"] = productClass
	}
	if alias != "" {
		filter["alias"] = alias
	}
	if model != "" {
		filter["model"] = model
	}
	if status != "" {
		fmtStatus, err := strconv.Atoi(status)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("Status must be an integer")
			return
		}
		filter["status"] = fmtStatus
	}

	devices, err := getDevices(w, filter, a.nc)
	if err != nil {
		log.Println("Error getting devices", err)
		return
	}

	if devices.Total == 0 {
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode("No devices found")
		if err != nil {
			log.Println(err)
		}
		return
	}

	if skip >= devices.Total {
		w.WriteHeader(http.StatusBadRequest)
		err := json.NewEncoder(w).Encode("Page number is out of range")
		if err != nil {
			log.Println(err)
		}
		return
	}

	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"pages":   devices.Total / page_size,
		"page":    page_number,
		"size":    page_size,
		"devices": devices.Devices,
		"total":   devices.Total,
	})
	if err != nil {
		log.Println(err)
	}
}

type DeviceAuth struct {
	User     string `json:"id"`
	Password string `json:"password"`
}

func (a *Api) deviceAuth(w http.ResponseWriter, r *http.Request) {

	user, err := a.db.FindUser(r.Context().Value("email").(string))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		utils.MarshallEncoder(err, w)
		return
	}
	if user.Level != db.AdminUser {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.Method == http.MethodGet {

		id := r.URL.Query().Get("id")
		if id != "" {
			entry, err := a.kv.Get(r.Context(), id)
			if err != nil {
				if err == jetstream.ErrKeyNotFound {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				utils.MarshallEncoder(err, w)
				return
			}
			utils.MarshallEncoder(map[string]string{
				id: string(entry.Value()),
			}, w)
			return
		}

		entries, err := a.kv.ListKeys(r.Context(), jetstream.IgnoreDeletes())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			utils.MarshallEncoder(err, w)
			return
		}

		listOfKeys := make(map[string]string)

		keys := entries.Keys()
		for key := range keys {
			entry, err := a.kv.Get(r.Context(), key)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				utils.MarshallEncoder(err, w)
				return
			}

			/*listOfKeys = append(listOfKeys, map[string]string{
				key: string(entry.Value()),
			})*/
			listOfKeys[key] = string(entry.Value())
		}

		utils.MarshallEncoder(listOfKeys, w)

	} else if r.Method == http.MethodDelete {

		id := r.URL.Query().Get("id")
		if id != "" {
			err := a.kv.Purge(r.Context(), id)
			if err != nil {
				if err == jetstream.ErrKeyNotFound {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				utils.MarshallEncoder(err, w)
				return
			}
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("No id provided", w)

	} else if r.Method == http.MethodPost {

		var deviceAuth DeviceAuth

		err := utils.MarshallDecoder(&deviceAuth, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			utils.MarshallEncoder(err, w)
			return
		}

		if deviceAuth.User != "" {
			_, err := a.kv.Get(r.Context(), deviceAuth.User)

			if err != nil {

				if err == jetstream.ErrKeyNotFound {
					_, err = a.kv.PutString(r.Context(), deviceAuth.User, deviceAuth.Password)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						utils.MarshallEncoder(err, w)
					}
					return
				}

				w.WriteHeader(http.StatusInternalServerError)
				utils.MarshallEncoder(err, w)
				return
			}

			w.WriteHeader(http.StatusConflict)
			utils.MarshallEncoder("Username already exists", w)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("device must have a user", w)

	} else {
		log.Println("Unknown method used in device auth api")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (a *Api) setDeviceAlias(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("No id provided", w)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("Error to decode payload: "+err.Error(), w)
		return
	}

	payloadLen := len(payload)
	if payloadLen == 0 {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("No alias provided", w)
		return
	}
	if payloadLen > 50 {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("Alias too long", w)
		return
	}

	_, err = bridge.NatsReq[[]byte](local.NATS_ADAPTER_SUBJECT+id+".device.alias", payload, w, a.nc)
	if err != nil {
		return
	}
}

func (a *Api) filterOptions(w http.ResponseWriter, r *http.Request) {

	resp, err := bridge.NatsReq[entity.FilterOptions](local.NATS_ADAPTER_SUBJECT+"devices.filterOptions", nil, w, a.nc)
	if err != nil {
		return
	}

	w.WriteHeader(resp.Code)
	w.Write(utils.Marshall(resp.Msg))
}

func (a *Api) updateTemplate(w http.ResponseWriter, r *http.Request) {

	name := r.URL.Query().Get("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("No name provided", w)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("Error to decode payload: "+err.Error(), w)
		return
	}

	payloadLen := len(payload)
	if payloadLen == 0 {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("No payload provided", w)
		return
	}

	err = a.db.UpdateTemplate(name, string(payload))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) addTemplate(w http.ResponseWriter, r *http.Request) {

	name := r.URL.Query().Get("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("No name provided", w)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("Error to decode payload: "+err.Error(), w)
		return
	}

	payloadLen := len(payload)
	if payloadLen == 0 {
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("No payload provided", w)
		return
	}

	vars := mux.Vars(r)
	switch vars["type"] {
	case "cwmp":
		err = a.db.AddTemplate(name, "cwmp", string(payload))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	case "usp":
		err = a.db.AddTemplate(name, "usp", string(payload))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		utils.MarshallEncoder("Invalid template type", w)
		return
	}
}

func (a *Api) getTemplate(w http.ResponseWriter, r *http.Request) {

	name := r.URL.Query().Get("name")
	msgType := r.URL.Query().Get("type")

	if name == "" {

		var filter bson.D
		if msgType == "" {
			filter = bson.D{}
		} else {
			filter = bson.D{{"type", msgType}}
		}

		result, err := a.db.AllTemplates(filter)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("Error to get all templates: " + err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
		return
	} else {
		t, err := a.db.FindTemplate(bson.D{{"name", name}})
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("error to find message: " + err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(t.Value))
		return
	}

}

func (a *Api) assignDeviceTenant(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value("role").(string)
	callerTenantID, _ := r.Context().Value("tenant_id").(string)

	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("No id provided")
		return
	}

	var body struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("Error to decode payload: " + err.Error())
		return
	}

	targetTenant := body.TenantID
	if targetTenant == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("tenant_id is required")
		return
	}

	// RBAC: super_admin can assign to any tenant; tenant_admin only to own tenant.
	// operator/viewer reach this handler because DeviceWritePermission allows devices:write,
	// but neither role holds that permission — they are rejected by middleware before here.
	switch callerRole {
	case db.RoleSuperAdmin:
		// allowed unrestricted
	case db.RoleTenantAdmin:
		if targetTenant != callerTenantID {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode("tenant_admin can only assign devices to own tenant")
			return
		}
	default:
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode("insufficient permissions to assign device tenant")
		return
	}

	// Verify target tenant exists
	if _, err := a.db.FindTenant(targetTenant); err != nil {
		if err == db.ErrorTenantNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode("tenant not found: " + targetTenant)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err.Error())
		}
		return
	}

	_, err := bridge.NatsReq[[]byte](local.NATS_ADAPTER_SUBJECT+id+".device.customer", []byte(targetTenant), w, a.nc)
	if err != nil {
		return
	}
}

func (a *Api) deleteTemplate(w http.ResponseWriter, r *http.Request) {

	name := r.URL.Query().Get("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("needs template name!")
		return
	} else {
		err := a.db.DeleteTemplate(name)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode("error to delete template: " + err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
