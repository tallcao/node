package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type LoraPanel struct {
	action *Payload_Metric

	guid string

	observer Observer

	Converter

	IHeart
}

func NewLoraPanel(guid string, c Converter, o Observer) *LoraPanel {

	item := &LoraPanel{
		action: &Payload_Metric{
			Name:      proto.String("action"),
			Datatype:  proto.Uint32(uint32(DataType_String)),
			Timestamp: proto.Uint64(0),
		},

		guid: guid,

		Converter: c,
		IHeart:    new(Heart),
		observer:  o,
	}

	return item
}

func (i *LoraPanel) Request(command string, params interface{}) {

}

func (i *LoraPanel) Response(data []byte) {

	if len(data) != 1 {
		return
	}

	v := ""

	switch data[0] {
	case 1:
		v = "1"
	case 2:
		v = "2"
	case 4:
		v = "3"
	}

	i.action.Value = &Payload_Metric_StringValue{v}
	*i.action.Timestamp = uint64(time.Now().UnixMicro())

	i.notifyAll()

}

func (i *LoraPanel) GetId() string {
	return i.guid
}

func (i *LoraPanel) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LORA_PANEL
}

func (i *LoraPanel) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.action)

	i.observer.Update(i.guid, p)

}

func (i *LoraPanel) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.action)

	return p

}

func (i *LoraPanel) HeartRequest() {

}
