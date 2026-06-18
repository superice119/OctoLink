package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	client        *mongo.Client
	users         *mongo.Collection
	template      *mongo.Collection
	tenants       *mongo.Collection
	roles         *mongo.Collection
	notifications *mongo.Collection
	ctx           context.Context
}

func NewDatabase(ctx context.Context, mongoUri string) Database {
	var db Database

	clientOptions := options.Client().ApplyURI(mongoUri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	db.client = client

	log.Println("Trying to ping Mongo database...")
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Couldn't connect to MongoDB --> ", err)
	}

	log.Println("Connected to MongoDB-->", mongoUri)

	db.ctx = ctx

	db.users = client.Database("account-mngr").Collection("users")
	indexField := bson.M{"email": 1}
	_, err = db.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    indexField,
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalln(err)
	}

	db.template = client.Database("general").Collection("templates")
	indexField = bson.M{"name": 1}
	_, err = db.template.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    indexField,
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalln(err)
	}

	db.tenants = client.Database("account-mngr").Collection("tenants")
	_, err = db.tenants.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"name": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalln(err)
	}

	db.roles = client.Database("account-mngr").Collection("roles")
	_, err = db.roles.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"name": 1, "tenant_id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatalln(err)
	}

	db.notifications = client.Database("octolink").Collection("notifications")
	_, err = db.notifications.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"device_sn": 1},
	})
	if err != nil {
		log.Println("Warning: failed to create notifications index:", err)
	}

	db.seedDefaultData()

	return db
}

// seedDefaultData ensures the default tenant and built-in roles exist.
func (d *Database) seedDefaultData() {
	// Default tenant
	err := d.CreateTenant(Tenant{
		ID:          DefaultTenantID,
		Name:        "Default",
		Description: "Default system tenant",
		CreatedAt:   time.Now().Unix(),
	})
	if err != nil && err != ErrorTenantExists {
		log.Println("Warning: could not seed default tenant:", err)
	}

	// Built-in system roles
	for name, perms := range BuiltinRolePermissions {
		role := Role{
			ID:          name,
			Name:        name,
			TenantID:    "",
			Permissions: perms,
			IsSystem:    true,
		}
		if err := d.CreateRole(role); err != nil && err != ErrorRoleExists {
			log.Println("Warning: could not seed role", name, ":", err)
		}
	}
}
