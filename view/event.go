package view

import (
	"edge/model"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type EventView struct {
	deviceType string

	dataCh chan<- *model.MqttMsg
}

func NewEventView(deviceType string, ch chan<- *model.MqttMsg) *EventView {

	return &EventView{
		deviceType: deviceType,
		dataCh:     ch,
	}
}

func (v *EventView) Update(uuid string, data map[string]any) {

	var tmp struct {
		DeviceUUID string `json:"device_uuid"`
		Timestamp  int64  `json:"timestamp"`

		Action any `json:"action"`
	}

	tmp.DeviceUUID = uuid
	tmp.Timestamp = time.Now().UnixMilli()

	if v, ok := data["action"]; ok {
		tmp.Action = v
	}

	payload, err := json.Marshal(tmp)
	if err != nil {
		log.Printf("ERROR: Failed to marshal action data: %v", err)
		return
	}

	msg := &model.MqttMsg{
		Topic:   fmt.Sprintf("controller/%v/%v", uuid, tmp.Action),
		Payload: payload,
	}

	v.dataCh <- msg
}
