package model

import (
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
)

type Door struct {
	on *Payload_Metric

	guid string

	observer Observer

	Converter

	IHeart
}

func NewDoor(guid string, c Converter, o Observer) *Door {

	item := &Door{

		on: &Payload_Metric{
			Name:      proto.String("on"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		Converter: c,

		IHeart: new(Heart),

		observer: o,
	}

	return item
}

func (i *Door) Request(command string, params interface{}) {

	switch command {
	case "on":
		i.SendFrame([]byte{0x01})
	case "off":
		i.SendFrame([]byte{0x00})
	case "toggle":
		if i.on.GetBooleanValue() {
			i.SendFrame([]byte{0x01})
		} else {
			i.SendFrame([]byte{0x00})
		}

	}

}
func (i *Door) UpdateDelta(c mqtt.Client, m mqtt.Message) {

	var update struct {
		On bool `json:"on"`
	}

	err := json.Unmarshal(m.Payload(), &update)

	if err != nil {
		log.Printf("ERROR: Failed to unmarshal door update delta: %v", err)
		return
	}

	currentOn := i.on.GetBooleanValue()
	if update.On != currentOn {
		switch update.On {
		case true:
			i.SendFrame([]byte{0x01})
		case false:
			i.SendFrame([]byte{0x00})
		}
	}
}
func (i *Door) Response(data []byte) {
	if len(data) != 8 && len(data) != 2 {
		return
	}

	on := i.on.GetBooleanValue()

	if len(data) == 8 {
		if data[0] == 0x00 {
			on = true
		}

		if data[0] == 0x01 {
			on = false

		}
	}

	if len(data) == 2 {
		if data[0] == 0x00 {
			on = false
		}

		if data[0] == 0x01 {
			on = true
		}
	}

	changed := (on != i.on.GetBooleanValue())

	i.on.Value = &Payload_Metric_BooleanValue{on}
	*i.on.Timestamp = uint64(time.Now().UnixMicro())

	if changed {

		i.notifyAll()
	}

}

func (i *Door) GetId() string {
	return i.guid
}

func (i *Door) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_DOOR
}

func (i *Door) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	i.observer.Update(i.guid, p)

}

func (i *Door) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	return p

}
