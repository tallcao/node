package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type Door struct {
	on *Payload_Metric

	guid string

	observerList []Observer

	converter Converter

	heart *Heart
}

func NewDoor(guid string, c Converter, o Observer) *Door {

	item := &Door{

		on: &Payload_Metric{
			Name:      proto.String("on"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		converter: c,

		heart: new(Heart),
	}

	item.register(o)

	return item
}

func (i *Door) Request(command string, params interface{}) {

	switch command {
	case "on":
		i.converter.SendFrame([]byte{0x01})
	case "off":
		i.converter.SendFrame([]byte{0x00})
	case "toggle":
		if i.on.GetBooleanValue() {
			i.converter.SendFrame([]byte{0x01})
		} else {
			i.converter.SendFrame([]byte{0x00})
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

func (i *Door) GetConverter() Converter {
	return i.converter
}

func (i *Door) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *Door) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *Door) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func (i *Door) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *Door) HeartCheck() {
	i.heart.HeartCheck()

}

func (i *Door) IsConnected() bool {
	return i.heart.Conected
}

func (i *Door) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *Door) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	return p

}
