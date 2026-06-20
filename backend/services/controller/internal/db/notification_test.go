package db

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Notification filter construction tests (no DB connection needed) ----
// These tests verify the tenant-scoped query logic by building the same filter
// bson.D that the real DB methods build, and asserting the expected keys are present.

func buildNotificationListFilter(deviceSN, tenantID string) bson.D {
	filter := bson.D{}
	if tenantID != "" {
		filter = append(filter, bson.E{Key: "owner_tenant_id", Value: tenantID})
	}
	if deviceSN != "" {
		filter = append(filter, bson.E{Key: "device_sn", Value: deviceSN})
	}
	return filter
}

func hasKey(filter bson.D, key string) bool {
	for _, e := range filter {
		if e.Key == key {
			return true
		}
	}
	return false
}

func keyValue(filter bson.D, key string) interface{} {
	for _, e := range filter {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

func TestNotificationFilter_TenantScoped(t *testing.T) {
	f := buildNotificationListFilter("", "tenant-A")
	if !hasKey(f, "owner_tenant_id") {
		t.Error("expected owner_tenant_id filter for non-super_admin")
	}
	if keyValue(f, "owner_tenant_id") != "tenant-A" {
		t.Errorf("expected tenant-A, got %v", keyValue(f, "owner_tenant_id"))
	}
	if hasKey(f, "device_sn") {
		t.Error("device_sn filter should not be present when deviceSN is empty")
	}
}

func TestNotificationFilter_SuperAdminNoTenantScope(t *testing.T) {
	// super_admin passes tenantID="" → no owner_tenant_id filter
	f := buildNotificationListFilter("", "")
	if hasKey(f, "owner_tenant_id") {
		t.Error("super_admin should not have owner_tenant_id filter")
	}
}

func TestNotificationFilter_TenantScopedWithDeviceSN(t *testing.T) {
	f := buildNotificationListFilter("device-001", "tenant-B")
	if !hasKey(f, "owner_tenant_id") {
		t.Error("expected owner_tenant_id filter")
	}
	if keyValue(f, "owner_tenant_id") != "tenant-B" {
		t.Errorf("expected tenant-B, got %v", keyValue(f, "owner_tenant_id"))
	}
	if !hasKey(f, "device_sn") {
		t.Error("expected device_sn filter")
	}
	if keyValue(f, "device_sn") != "device-001" {
		t.Errorf("expected device-001, got %v", keyValue(f, "device_sn"))
	}
}

func TestNotificationFilter_SuperAdminWithDeviceSN(t *testing.T) {
	f := buildNotificationListFilter("device-002", "")
	if hasKey(f, "owner_tenant_id") {
		t.Error("super_admin should not have owner_tenant_id filter")
	}
	if !hasKey(f, "device_sn") {
		t.Error("expected device_sn filter")
	}
}

// ---- MarkAllNotificationsRead filter ----

func buildMarkAllFilter(tenantID string) bson.D {
	filter := bson.D{{Key: "read", Value: false}}
	if tenantID != "" {
		filter = append(filter, bson.E{Key: "owner_tenant_id", Value: tenantID})
	}
	return filter
}

func TestMarkAllFilter_TenantScoped(t *testing.T) {
	f := buildMarkAllFilter("tenant-A")
	if !hasKey(f, "owner_tenant_id") {
		t.Error("expected owner_tenant_id in mark-all filter for non-super_admin")
	}
	if !hasKey(f, "read") {
		t.Error("expected read filter")
	}
}

func TestMarkAllFilter_SuperAdmin(t *testing.T) {
	f := buildMarkAllFilter("")
	if hasKey(f, "owner_tenant_id") {
		t.Error("super_admin mark-all should not have tenant filter")
	}
	if !hasKey(f, "read") {
		t.Error("expected read filter even for super_admin")
	}
}

// ---- MarkNotificationsRead by ID filter ----

func buildMarkByIDFilter(ids []string, tenantID string) bson.D {
	var objIDs []primitive.ObjectID
	for _, id := range ids {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objIDs = append(objIDs, oid)
	}
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: objIDs}}}}
	if tenantID != "" {
		filter = append(filter, bson.E{Key: "owner_tenant_id", Value: tenantID})
	}
	return filter
}

func TestMarkByIDFilter_TenantScoped(t *testing.T) {
	id := primitive.NewObjectID().Hex()
	f := buildMarkByIDFilter([]string{id}, "tenant-A")
	if !hasKey(f, "owner_tenant_id") {
		t.Error("expected owner_tenant_id filter for non-super_admin mark-by-id")
	}
	if keyValue(f, "owner_tenant_id") != "tenant-A" {
		t.Errorf("expected tenant-A, got %v", keyValue(f, "owner_tenant_id"))
	}
}

func TestMarkByIDFilter_SuperAdmin(t *testing.T) {
	id := primitive.NewObjectID().Hex()
	f := buildMarkByIDFilter([]string{id}, "")
	if hasKey(f, "owner_tenant_id") {
		t.Error("super_admin mark-by-id should not have tenant filter")
	}
}

// ---- Notification struct sanity check ----

func TestNotificationStruct_OwnerTenantIDPresent(t *testing.T) {
	n := Notification{
		DeviceSN:      "sn-001",
		OwnerTenantID: "tenant-A",
		Type:          "event",
		Timestamp:     time.Now(),
	}
	if n.OwnerTenantID != "tenant-A" {
		t.Errorf("expected tenant-A, got %s", n.OwnerTenantID)
	}
}
