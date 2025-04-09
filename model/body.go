package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type BodySensor struct {
	body *Payload_Metric

	guid string

	observerList []Observer

	converter Converter

	heart *Heart
}

func NewBodySensor(guid string, c Converter, o Observer) *BodySensor {

	item := &BodySensor{

		body: &Payload_Metric{
			Name:      proto.String("body"),
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

func (i *BodySensor) GetConverter() Converter {
	return i.converter
}

func (i *BodySensor) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *BodySensor) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *BodySensor) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.body)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func (i *BodySensor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *BodySensor) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *BodySensor) HeartCheck() {
	i.heart.HeartCheck()
}

func (i *BodySensor) IsConnected() bool {
	return i.heart.Conected
}

func (i *BodySensor) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *BodySensor) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.body)

	return p

}
