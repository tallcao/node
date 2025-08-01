package view

import (
	"edge/model"
	"encoding/json"
	"fmt"
	"log"
)

type ShadowView struct {
	node string

	dataCh chan<- *model.MqttMsg
}

func NewShadowView(node string, ch chan<- *model.MqttMsg) *ShadowView {

	return &ShadowView{
		node:   node,
		dataCh: ch,
	}
}

func (v *ShadowView) Update(uuid string, state map[string]any) {

	var data struct {
		DeviceUUID string         `json:"device_uuid"`
		State      map[string]any `json:"state"`
	}

	data.DeviceUUID = uuid
	data.State = state

	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("ERROR: Failed to marshal shadow data: %v", err)
		return
	}

	msg := &model.MqttMsg{
		Topic:   fmt.Sprintf("%v/shadow/update/reported", uuid),
		Payload: payload,
	}

	v.dataCh <- msg
}
