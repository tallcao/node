package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type EValve struct {
	on *Payload_Metric

	guid string

	observer Observer

	Converter

	IHeart
}

func NewEValve(guid string, c Converter, o Observer) *EValve {

	item := &EValve{
		on: &Payload_Metric{
			Name:      proto.String("on"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		Converter: c,

		IHeart:   new(Heart),
		observer: o,
	}

	return item
}

func (i *EValve) Request(command string, params interface{}) {

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
	case "getStatus":
		i.SendFrame([]byte{0x02})
	}
}

func (i *EValve) Response(data []byte) {

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
		if data[1] == 0x00 {
			on = true
		}

		if data[1] == 0x01 {
			on = false
		}
	}

	changed := (on != i.on.GetBooleanValue())

	i.on.Value = &Payload_Metric_BooleanValue{on}
	*i.on.Timestamp = uint64(time.Now().UnixMicro())

	if changed {

		i.notifyAll()
	}

}

func (i *EValve) GetId() string {
	return i.guid
}
func (i *EValve) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_E_VALVE
}

func (i *EValve) notifyAll() {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	i.observer.Update(i.guid, p)

}

func (i *EValve) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (i *EValve) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	return p

}
