package db

import (
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Notification struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	DeviceSN       string             `json:"device_sn" bson:"device_sn"`
	SubscriptionID string             `json:"subscription_id" bson:"subscription_id"`
	Type           string             `json:"type" bson:"type"`
	ObjPath        string             `json:"obj_path,omitempty" bson:"obj_path,omitempty"`
	EventName      string             `json:"event_name,omitempty" bson:"event_name,omitempty"`
	Params         map[string]string  `json:"params,omitempty" bson:"params,omitempty"`
	ParamPath      string             `json:"param_path,omitempty" bson:"param_path,omitempty"`
	ParamValue     string             `json:"param_value,omitempty" bson:"param_value,omitempty"`
	Read           bool               `json:"read" bson:"read"`
	Timestamp      time.Time          `json:"timestamp" bson:"timestamp"`
}

type NotificationsList struct {
	Notifications []Notification `json:"notifications"`
	Total         int64          `json:"total"`
	Unread        int64          `json:"unread"`
}

func (d *Database) CreateNotification(n Notification) error {
	n.ID = primitive.NewObjectID()
	if n.Timestamp.IsZero() {
		n.Timestamp = time.Now().UTC()
	}
	_, err := d.notifications.InsertOne(d.ctx, n)
	if err != nil {
		log.Printf("Failed to insert notification: %v", err)
	}
	return err
}

func (d *Database) RetrieveNotifications(page, pageSize int64, deviceSN string) (*NotificationsList, error) {
	filter := bson.D{}
	if deviceSN != "" {
		filter = bson.D{{Key: "device_sn", Value: deviceSN}}
	}

	total, err := d.notifications.CountDocuments(d.ctx, filter)
	if err != nil {
		return nil, err
	}

	unreadFilter := append(bson.D{}, filter...)
	unreadFilter = append(unreadFilter, bson.E{Key: "read", Value: false})
	unread, err := d.notifications.CountDocuments(d.ctx, unreadFilter)
	if err != nil {
		return nil, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)

	cursor, err := d.notifications.Find(d.ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(d.ctx)

	var notifications []Notification
	if err := cursor.All(d.ctx, &notifications); err != nil {
		return nil, err
	}
	if notifications == nil {
		notifications = []Notification{}
	}

	return &NotificationsList{
		Notifications: notifications,
		Total:         total,
		Unread:        unread,
	}, nil
}

func (d *Database) MarkNotificationsRead(ids []string) error {
	var objIDs []primitive.ObjectID
	for _, id := range ids {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		objIDs = append(objIDs, oid)
	}
	if len(objIDs) == 0 {
		return nil
	}
	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: objIDs}}}}
	_, err := d.notifications.UpdateMany(d.ctx, filter, bson.D{{Key: "$set", Value: bson.D{{Key: "read", Value: true}}}})
	return err
}

func (d *Database) MarkAllNotificationsRead() error {
	_, err := d.notifications.UpdateMany(d.ctx, bson.D{{Key: "read", Value: false}}, bson.D{{Key: "$set", Value: bson.D{{Key: "read", Value: true}}}})
	return err
}

func (d *Database) DeleteNotifications(deviceSN string) (int64, error) {
	filter := bson.D{}
	if deviceSN != "" {
		filter = bson.D{{Key: "device_sn", Value: deviceSN}}
	}
	result, err := d.notifications.DeleteMany(d.ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}
