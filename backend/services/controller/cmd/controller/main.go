package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/leandrofars/oktopus/internal/api"
	"github.com/leandrofars/oktopus/internal/api/auth"
	"github.com/leandrofars/oktopus/internal/bridge"
	"github.com/leandrofars/oktopus/internal/config"
	"github.com/leandrofars/oktopus/internal/db"
	"github.com/leandrofars/oktopus/internal/nats"
)

func main() {
	done := make(chan os.Signal, 1)

	signal.Notify(done, syscall.SIGINT)

	// Fail closed if the JWT signing secret is missing/default/weak, so the
	// controller never runs with a forgeable token key (WS-38).
	if err := auth.RequireSecret(); err != nil {
		log.Fatalf("insecure JWT configuration: %v", err)
	}

	c := config.NewConfig()

	js, nc, kv := nats.StartNatsClient(c.Nats)

	bridge := bridge.NewBridge(js, nc)

	db := db.NewDatabase(c.Mongo.Ctx, c.Mongo.Uri)

	restAPI := api.NewApi(c, js, nc, bridge, db, kv)
	restAPI.StartApi()
	api.StartNotificationSubscriber(nc, db)

	<-done

	log.Println("rest api is shutting down...")
}
