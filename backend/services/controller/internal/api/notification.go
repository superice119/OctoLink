package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/leandrofars/oktopus/internal/db"
	"github.com/leandrofars/oktopus/internal/utils"
	"github.com/nats-io/nats.go"
)

// StartNotificationSubscriber subscribes to NATS notification.v1.> and persists events.
func StartNotificationSubscriber(nc *nats.Conn, database db.Database) {
	_, err := nc.Subscribe("notification.v1.>", func(msg *nats.Msg) {
		var n db.Notification
		if err := json.Unmarshal(msg.Data, &n); err != nil {
			log.Printf("NotificationSubscriber: failed to unmarshal: %v", err)
			return
		}
		if err := database.CreateNotification(n); err != nil {
			log.Printf("NotificationSubscriber: failed to store notification: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("NotificationSubscriber: failed to subscribe: %v", err)
	}
	log.Println("NotificationSubscriber: listening on notification.v1.>")
}

func (a *Api) listNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	deviceSN := r.URL.Query().Get("device_sn")

	page, _ := strconv.ParseInt(pageStr, 10, 64)
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.ParseInt(pageSizeStr, 10, 64)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, err := a.db.RetrieveNotifications(page, pageSize, deviceSN)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(utils.Marshall(err.Error()))
		return
	}
	utils.MarshallEncoder(list, w)
}

func (a *Api) markNotificationsRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var body struct {
		IDs []string `json:"ids"`
		All bool     `json:"all"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	var err error
	if body.All || len(body.IDs) == 0 {
		err = a.db.MarkAllNotificationsRead()
	} else {
		err = a.db.MarkNotificationsRead(body.IDs)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(utils.Marshall(err.Error()))
		return
	}
	w.Write(utils.Marshall("ok"))
}

func (a *Api) deleteNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	deviceSN := r.URL.Query().Get("device_sn")
	count, err := a.db.DeleteNotifications(deviceSN)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(utils.Marshall(err.Error()))
		return
	}
	utils.MarshallEncoder(map[string]int64{"deleted": count}, w)
}
