package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type EValve struct {
	on *Payload_Metric

	guid string

	observerList []Observer

	converter Converter

	heart *Heart
}

func NewEValve(guid string, c Converter, o Observer) *EValve {

	item := &EValve{
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

func (i *EValve) Request(command string, params interface{}) {

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
	case "getStatus":
		i.converter.SendFrame([]byte{0x02})
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
func (i *EValve) GetConverter() Converter {
	return i.converter
}

func (i *EValve) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *EValve) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *EValve) notifyAll() {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func (i *EValve) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *EValve) HeartCheck() {
	i.heart.HeartCheck()
	if i.heart.Conected && i.heart.Changed() {
		i.Request("getStatus", nil)
	}
}

func (i *EValve) IsConnected() bool {
	return i.heart.Conected
}

func (i *EValve) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *EValve) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	return p

}
