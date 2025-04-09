package model

import (
	"edge/utils"
	"time"

	"google.golang.org/protobuf/proto"
)

type RainSensor struct {
	raining *Payload_Metric

	guid string

	observerList []Observer

	converter Converter

	PassiveReporting *PassiveReporting
	heart            *Heart
}

func NewRainSensor(guid string, c Converter, o Observer) *RainSensor {

	item := &RainSensor{
		raining: &Payload_Metric{
			Name:      proto.String("raining"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		converter: c,

		PassiveReporting: &PassiveReporting{
			Interval: 60 * 5,
		},

		heart: new(Heart),
	}

	item.register(o)

	return item
}

func (i *RainSensor) Request(command string, params interface{}) {

	var data []byte

	switch command {

	case "getStatus":
		data = []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x01}
	case "setInterval":
		i.PassiveReporting.SetInterval(params)

		return
	default:
		return
	}

	crc, err := utils.CRC16(data)
	if err != nil {
		return
	}

	data = append(data, crc...)
	i.converter.SendFrame(data)

}

func (i *RainSensor) Response(data []byte) {
	if len(data) < 7 {
		return
	}

	raining := i.raining.GetBooleanValue()
	if data[4] == 0x00 {
		raining = false
	}

	if data[4] == 0x01 {
		raining = true
	}

	changed := (raining != i.raining.GetBooleanValue())

	i.raining.Value = &Payload_Metric_BooleanValue{raining}
	*i.raining.Timestamp = uint64(time.Now().UnixMicro())

	if changed {

		i.notifyAll()
	}

}

func (i *RainSensor) GetId() string {
	return i.guid
}
func (i *RainSensor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_RAIN
}

func (i *RainSensor) GetConverter() Converter {
	return i.converter
}

func (i *RainSensor) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *RainSensor) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *RainSensor) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.raining)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func (i *RainSensor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 4800, 0, 8, 1
}

func (i *RainSensor) StartLoopRequest() {
	i.PassiveReporting.StartLoopRequest(i.Request, "getStatus")
}

func (i *RainSensor) StopLoopRequest() {
	i.PassiveReporting.StopLoopRequest()
}

func (i *RainSensor) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *RainSensor) HeartCheck() {
	i.heart.HeartCheck()
	if i.heart.Conected && i.heart.Changed() {
		i.Request("getStatus", nil)
	}
}

func (i *RainSensor) IsConnected() bool {
	return i.heart.Conected
}

func (i *RainSensor) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *RainSensor) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.raining)

	return p

}
