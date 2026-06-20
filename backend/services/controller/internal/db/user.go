package db

import (
	"errors"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type UserLevels int32

const (
	NormalUser UserLevels = iota
	AdminUser
)

type User struct {
	Email    string     `json:"email"              bson:"email"`
	Name     string     `json:"name"               bson:"name"`
	Password string     `json:"password,omitempty" bson:"password"`
	Level    UserLevels `json:"level"              bson:"level"`
	Phone    string     `json:"phone"              bson:"phone"`
	TenantID string     `json:"tenant_id,omitempty" bson:"tenant_id,omitempty"`
	Role     string     `json:"role,omitempty"      bson:"role,omitempty"`
}

// EffectiveTenantID returns the user's tenant_id, falling back to DefaultTenantID
// for legacy users or users created without an explicit tenant.
// super_admin is not tenant-scoped; for all other roles an empty tenant_id is
// replaced with the default tenant so JWT claims are never empty.
func (u *User) EffectiveTenantID() string {
	if u.TenantID != "" {
		return u.TenantID
	}
	if u.EffectiveRole() == RoleSuperAdmin {
		return u.TenantID // super_admin may legitimately have no tenant
	}
	return DefaultTenantID
}

// IsGlobalRole reports whether role is a global (cross-tenant) role that only
// super_admin is allowed to grant. tenant_admin may never elevate a user to one
// of these roles.
func IsGlobalRole(role string) bool {
	return role == RoleSuperAdmin
}
func (u *User) EffectiveRole() string {
	if u.Role != "" {
		return u.Role
	}
	if u.Level == AdminUser {
		return RoleSuperAdmin
	}
	return RoleOperator
}

var ErrorUserExists = errors.New("User already exists")

func (d *Database) RegisterUser(user User) error {
	err := d.users.FindOne(d.ctx, bson.D{{"email", user.Email}}).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err = d.users.InsertOne(d.ctx, user)
			return err
		}
		log.Println(err)
		return err
	} else {
		return ErrorUserExists
	}
}

func (d *Database) UpdatePassword(user User) error {
	_, err := d.users.UpdateOne(d.ctx, bson.D{{"email", user.Email}}, bson.D{{"$set", bson.D{{"password", user.Password}}}})
	return err
}

func (d *Database) FindAllUsers() ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	cursor, err := d.users.Find(d.ctx, bson.D{{}})
	if err != nil {
		return []map[string]interface{}{}, err
	}
	if err = cursor.All(d.ctx, &result); err != nil {
		log.Fatal(err)
	}
	return result, err
}

func (d *Database) FindUsersByTenant(tenantID string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	cursor, err := d.users.Find(d.ctx, bson.D{{Key: "tenant_id", Value: tenantID}})
	if err != nil {
		return []map[string]interface{}{}, err
	}
	if err = cursor.All(d.ctx, &result); err != nil {
		return []map[string]interface{}{}, err
	}
	return result, nil
}

func (d *Database) UpdateUserRole(email, role, tenantID string) error {
	_, err := d.users.UpdateOne(
		d.ctx,
		bson.D{{Key: "email", Value: email}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "role", Value: role},
			{Key: "tenant_id", Value: tenantID},
		}}},
	)
	return err
}

func (d *Database) FindUser(email string) (User, error) {
	var result User
	err := d.users.FindOne(d.ctx, bson.D{{"email", email}}).Decode(&result)
	return result, err
}

func (d *Database) DeleteUser(email string) error {
	_, err := d.users.DeleteOne(d.ctx, bson.D{{"email", email}})
	return err
}

func (user *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	user.Password = string(bytes)
	return nil
}

func (user *User) CheckPassword(providedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(providedPassword))
	if err != nil {
		return err
	}
	return nil
}
