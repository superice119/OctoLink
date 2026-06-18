package usp_handler

import (
	"encoding/json"
	"log"
	"time"

	"github.com/OktopUSP/oktopus/backend/services/mtp/adapter/internal/usp"
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

func (h *Handler) HandleNotify(device, subject string, data []byte, mtpName string, ack func()) {
	defer ack()
	log.Printf("Device %s sent USP Notify, subject: %s", device, subject)

	var record usp_record.Record
	if err := proto.Unmarshal(data, &record); err != nil {
		log.Printf("HandleNotify: failed to unmarshal USP Record: %v", err)
		return
	}

	// Extract USP Msg payload from whichever record context the agent used.
	var msgPayload []byte
	switch {
	case record.GetNoSessionContext() != nil:
		msgPayload = record.GetNoSessionContext().GetPayload()
	case record.GetSessionContext() != nil:
		// SessionContextRecord.Payload is [][]byte (segmented); concatenate.
		for _, chunk := range record.GetSessionContext().GetPayload() {
			msgPayload = append(msgPayload, chunk...)
		}
	default:
		log.Printf("HandleNotify: unsupported record type for device %s", device)
		return
	}

	var message usp_msg.Msg
	if err := proto.Unmarshal(msgPayload, &message); err != nil {
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

	// Send NotifyResp when the agent requests a response (send_resp=true).
	if notify.SendResp {
		h.sendNotifyResponse(device, message.Header.MsgId, notify.SubscriptionId, mtpName)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("HandleNotify: failed to marshal notification: %v", err)
		return
	}

	subjectOut := NotificationSubjectPrefix + device
	if err := h.nc.Publish(subjectOut, payload); err != nil {
		log.Printf("HandleNotify: failed to publish notification: %v", err)
	} else {
		log.Printf("HandleNotify: published notification to %s", subjectOut)
	}
}

// sendNotifyResponse builds a USP NotifyResp and sends it back to the device
// via the same MTP adapter that delivered the original Notify.
func (h *Handler) sendNotifyResponse(device, msgID, subscriptionID, mtpName string) {
	respMsg := usp_msg.Msg{
		Header: &usp_msg.Header{
			MsgId:   msgID,
			MsgType: usp_msg.Header_NOTIFY_RESP,
		},
		Body: &usp_msg.Body{
			MsgBody: &usp_msg.Body_Response{
				Response: &usp_msg.Response{
					RespType: &usp_msg.Response_NotifyResp{
						NotifyResp: &usp_msg.NotifyResp{
							SubscriptionId: subscriptionID,
						},
					},
				},
			},
		},
	}

	protoMsg, err := proto.Marshal(&respMsg)
	if err != nil {
		log.Printf("sendNotifyResponse: marshal error: %v", err)
		return
	}

	record := usp.NewUspRecord(protoMsg, device, h.cid)
	protoRecord, err := proto.Marshal(&record)
	if err != nil {
		log.Printf("sendNotifyResponse: marshal record error: %v", err)
		return
	}

	// Publish on the adapter-specific .api topic: the MTP adapter (ws/mqtt/stomp)
	// subscribes to {mtp}-adapter.usp.v1.*.api and forwards the payload to the device.
	topic := mtpName + "-adapter.usp.v1." + device + ".api"
	if err := h.nc.Publish(topic, protoRecord); err != nil {
		log.Printf("sendNotifyResponse: publish error: %v", err)
	} else {
		log.Printf("sendNotifyResponse: sent NotifyResp to %s (subscriptionId=%s)", device, subscriptionID)
	}
}
