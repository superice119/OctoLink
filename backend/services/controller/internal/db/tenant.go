package db

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Tenant represents an isolated organisational unit.
type Tenant struct {
	ID          string `json:"id"          bson:"_id"`
	Name        string `json:"name"        bson:"name"`
	Description string `json:"description" bson:"description"`
	CreatedAt   int64  `json:"created_at"  bson:"created_at"`
}

var (
	ErrorTenantExists   = errors.New("tenant already exists")
	ErrorTenantNotFound = errors.New("tenant not found")
)

// DefaultTenantID is the catch-all tenant for pre-existing (non-migrated) data.
const DefaultTenantID = "default"

func (d *Database) CreateTenant(t Tenant) error {
	if t.ID == "" {
		return errors.New("tenant id is required")
	}
	t.CreatedAt = time.Now().Unix()
	err := d.tenants.FindOne(d.ctx, bson.D{{Key: "_id", Value: t.ID}}).Err()
	if err == nil {
		return ErrorTenantExists
	}
	if err != mongo.ErrNoDocuments {
		return err
	}
	_, err = d.tenants.InsertOne(d.ctx, t)
	return err
}

func (d *Database) FindTenant(id string) (Tenant, error) {
	var t Tenant
	err := d.tenants.FindOne(d.ctx, bson.D{{Key: "_id", Value: id}}).Decode(&t)
	if err == mongo.ErrNoDocuments {
		return t, ErrorTenantNotFound
	}
	return t, err
}

func (d *Database) FindAllTenants() ([]Tenant, error) {
	var tenants []Tenant
	cursor, err := d.tenants.Find(d.ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(d.ctx, &tenants); err != nil {
		return nil, err
	}
	return tenants, nil
}

func (d *Database) UpdateTenant(t Tenant) error {
	res, err := d.tenants.UpdateOne(
		d.ctx,
		bson.D{{Key: "_id", Value: t.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: t.Name},
			{Key: "description", Value: t.Description},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrorTenantNotFound
	}
	return nil
}

func (d *Database) DeleteTenant(id string) error {
	res, err := d.tenants.DeleteOne(d.ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrorTenantNotFound
	}
	return nil
}
