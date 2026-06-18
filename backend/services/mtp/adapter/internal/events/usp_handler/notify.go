package usp_handler

import (
	"encoding/json"
	"log"
	"time"

	"github.com/OktopUSP/oktopus/backend/services/mtp/adapter/internal/usp/usp_msg"
	"github.com/OktopUSP/oktopus/backend/services/mtp/adapter/internal/usp/usp_record"
	"google.golang.org/protobuf/proto"
)

const NotificationSubjectPrefix = "notification.v1."

type NotificationEvent struct {
	DeviceSN       string            `json:"device_sn"`
	SubscriptionID string            `json:"subscription_id"`
	Type           string            `json:"type"`
	ObjPath        string            `json:"obj_path,omitempty"`
	EventName      string            `json:"event_name,omitempty"`
	Params         map[string]string `json:"params,omitempty"`
	ParamPath      string            `json:"param_path,omitempty"`
	ParamValue     string            `json:"param_value,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

func (h *Handler) HandleNotify(device, subject string, data []byte, ack func()) {
	defer ack()
	log.Printf("Device %s sent USP Notify, subject: %s", device, subject)

	var record usp_record.Record
	err := proto.Unmarshal(data, &record)
	if err != nil {
		log.Printf("HandleNotify: failed to unmarshal USP Record: %v", err)
		return
	}

	var message usp_msg.Msg
	err = proto.Unmarshal(record.GetNoSessionContext().Payload, &message)
	if err != nil {
		log.Printf("HandleNotify: failed to unmarshal USP Msg: %v", err)
		return
	}

	notify := message.Body.GetRequest().GetNotify()
	if notify == nil {
		log.Printf("HandleNotify: message does not contain Notify request")
		return
	}

	event := NotificationEvent{
		DeviceSN:       device,
		SubscriptionID: notify.SubscriptionId,
		Timestamp:      time.Now().UTC(),
	}

	switch n := notify.Notification.(type) {
	case *usp_msg.Notify_Event_:
		event.Type = "event"
		event.ObjPath = n.Event.ObjPath
		event.EventName = n.Event.EventName
		event.Params = n.Event.Params
	case *usp_msg.Notify_ValueChange_:
		event.Type = "value_change"
		event.ParamPath = n.ValueChange.ParamPath
		event.ParamValue = n.ValueChange.ParamValue
	case *usp_msg.Notify_ObjCreation:
		event.Type = "obj_creation"
		event.ObjPath = n.ObjCreation.ObjPath
	case *usp_msg.Notify_ObjDeletion:
		event.Type = "obj_deletion"
		event.ObjPath = n.ObjDeletion.ObjPath
	case *usp_msg.Notify_OperComplete:
		event.Type = "oper_complete"
		event.ObjPath = n.OperComplete.ObjPath
	case *usp_msg.Notify_OnBoardReq:
		event.Type = "on_board_req"
	default:
		event.Type = "unknown"
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("HandleNotify: failed to marshal notification: %v", err)
		return
	}

	subjectOut := NotificationSubjectPrefix + device
	err = h.nc.Publish(subjectOut, payload)
	if err != nil {
		log.Printf("HandleNotify: failed to publish notification: %v", err)
	} else {
		log.Printf("HandleNotify: published notification to %s", subjectOut)
	}
}
