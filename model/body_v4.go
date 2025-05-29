package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type BodySensorV4 struct {
	body *Payload_Metric

	guid string

	observer Observer

	Converter

	IHeart
}

func NewBodySensorV4(guid string, c Converter, o Observer) *BodySensorV4 {

	item := &BodySensorV4{
		body: &Payload_Metric{
			Name:      proto.String("body"),
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

func (i *BodySensorV4) Request(command string, params interface{}) {

}

func (i *BodySensorV4) Response(data []byte) {

	if len(data) < 8 {
		return
	}
	body := i.body.GetBooleanValue()

	if data[6] == 0xAA {
		body = true
	}

	if data[6] == 0x55 {
		body = false
	}

	changed := (body != i.body.GetBooleanValue())

	i.body.Value = &Payload_Metric_BooleanValue{body}
	*i.body.Timestamp = uint64(time.Now().UnixMicro())

	if changed {

		i.notifyAll()
	}

}

func (i *BodySensorV4) GetId() string {
	return i.guid
}

func (i *BodySensorV4) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BODY_V4
}

func (i *BodySensorV4) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.body)

	i.observer.Update(i.guid, p)

}

func (i *BodySensorV4) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *BodySensorV4) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.body)

	return p

}
