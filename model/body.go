package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type BodySensor struct {
	body *Payload_Metric

	guid string

	observer Observer

	Converter

	IHeart
}

func NewBodySensor(guid string, c Converter, o Observer) *BodySensor {

	item := &BodySensor{

		body: &Payload_Metric{
			Name:      proto.String("body"),
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

func (i *BodySensor) Request(command string, params interface{}) {

}

func (i *BodySensor) Response(data []byte) {

	if len(data) < 7 {
		return
	}
	body := i.body.GetBooleanValue()

	if data[4] == 0x00 {
		body = false
	}

	if data[4] == 0x01 {
		body = true
	}

	changed := (body != i.body.GetBooleanValue())

	i.body.Value = &Payload_Metric_BooleanValue{body}
	*i.body.Timestamp = uint64(time.Now().UnixMicro())

	if changed {

		i.notifyAll()
	}

}

func (i *BodySensor) GetId() string {
	return i.guid
}

func (i *BodySensor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BODY
}

func (i *BodySensor) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.body)

	i.observer.Update(i.guid, p)

}

func (i *BodySensor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *BodySensor) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.body)

	return p

}
