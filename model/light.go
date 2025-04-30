package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type Light struct {
	on *Payload_Metric

	guid string

	observer Observer

	converter Converter

	heart *Heart
}

func NewLight(guid string, c Converter, o Observer) *Light {

	item := &Light{
		on: &Payload_Metric{
			Name:      proto.String("on"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},
		guid:      guid,
		converter: c,

		heart: new(Heart),

		observer: o,
	}

	return item
}

func (i *Light) Request(command string, params interface{}) {

	switch command {
	case "on":
		i.converter.SendFrame([]byte{0x01})
	case "off":
		i.converter.SendFrame([]byte{0x00})
	case "toggle":
		if i.on.GetBooleanValue() {
			i.converter.SendFrame([]byte{0x00})
		} else {
			i.converter.SendFrame([]byte{0x01})
		}
	}

}

func (i *Light) Response(data []byte) {

	if len(data) != 8 && len(data) != 2 {
		return
	}

	on := i.on.GetBooleanValue()

	if len(data) == 8 {
		if data[0] == 0x00 {
			on = false
		}

		if data[0] == 0x01 {
			on = true
		}
	}

	if len(data) == 2 {
		if data[1] == 0x00 {
			on = false

		}

		if data[1] == 0x01 {
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

// func (i *Light) Turn(on bool) {

// 	if on {
// 		i.on()
// 	} else {
// 		i.off()
// 	}
// }
// func (i *Light) on() {
// 	i.converter.SendFrame([]byte{0x01})
// }

// func (i *Light) off() {
// 	i.converter.SendFrame([]byte{0x00})
// }

func (i *Light) GetId() string {
	return i.guid
}

func (i *Light) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_LIGHT
}

func (i *Light) GetConverter() Converter {
	return i.converter
}

func (i *Light) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.on)

	i.observer.Update(i.guid, p)
}

func (i *Light) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *Light) HeartCheck() {
	i.heart.HeartCheck()

}

func (i *Light) IsConnected() bool {
	return i.heart.Conected
}

func (i *Light) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *Light) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.on)

	return p

}
