package db

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Built-in role names.
const (
	RoleSuperAdmin  = "super_admin"
	RoleTenantAdmin = "tenant_admin"
	RoleOperator    = "operator"
	RoleViewer      = "viewer"
)

// Permission constants.
const (
	PermDevicesRead   = "devices:read"
	PermDevicesWrite  = "devices:write"
	PermUsersRead     = "users:read"
	PermUsersWrite    = "users:write"
	PermTenantsManage = "tenants:manage"
	PermRolesManage   = "roles:manage"
)

// BuiltinRolePermissions maps built-in role names to their permission sets.
var BuiltinRolePermissions = map[string][]string{
	RoleSuperAdmin:  {PermDevicesRead, PermDevicesWrite, PermUsersRead, PermUsersWrite, PermTenantsManage, PermRolesManage},
	RoleTenantAdmin: {PermDevicesRead, PermDevicesWrite, PermUsersRead, PermUsersWrite},
	RoleOperator:    {PermDevicesRead, PermDevicesWrite},
	RoleViewer:      {PermDevicesRead},
}

// HasPermission reports whether the given role has the required permission.
// Custom roles stored in the DB are looked up via the Database method.
func HasPermission(role, permission string) bool {
	perms, ok := BuiltinRolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// Role is a named set of permissions, optionally scoped to a tenant.
type Role struct {
	ID          string   `json:"id"          bson:"_id"`
	Name        string   `json:"name"        bson:"name"`
	TenantID    string   `json:"tenant_id"   bson:"tenant_id"`
	Permissions []string `json:"permissions" bson:"permissions"`
	IsSystem    bool     `json:"is_system"   bson:"is_system"`
}

var (
	ErrorRoleExists   = errors.New("role already exists")
	ErrorRoleNotFound = errors.New("role not found")
)

func (d *Database) CreateRole(r Role) error {
	if r.ID == "" {
		return errors.New("role id is required")
	}
	err := d.roles.FindOne(d.ctx, bson.D{{Key: "_id", Value: r.ID}}).Err()
	if err == nil {
		return ErrorRoleExists
	}
	if err != mongo.ErrNoDocuments {
		return err
	}
	_, err = d.roles.InsertOne(d.ctx, r)
	return err
}

func (d *Database) FindRole(id string) (Role, error) {
	var r Role
	err := d.roles.FindOne(d.ctx, bson.D{{Key: "_id", Value: id}}).Decode(&r)
	if err == mongo.ErrNoDocuments {
		return r, ErrorRoleNotFound
	}
	return r, err
}

func (d *Database) FindAllRoles(tenantID string) ([]Role, error) {
	filter := bson.D{}
	if tenantID != "" {
		filter = bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "tenant_id", Value: tenantID}},
			bson.D{{Key: "is_system", Value: true}},
		}}}
	}
	var roles []Role
	cursor, err := d.roles.Find(d.ctx, filter)
	if err != nil {
		return nil, err
	}
	if err = cursor.All(d.ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (d *Database) UpdateRole(r Role) error {
	res, err := d.roles.UpdateOne(
		d.ctx,
		bson.D{{Key: "_id", Value: r.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: r.Name},
			{Key: "permissions", Value: r.Permissions},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrorRoleNotFound
	}
	return nil
}

func (d *Database) DeleteRole(id string) error {
	res, err := d.roles.DeleteOne(d.ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrorRoleNotFound
	}
	return nil
}
