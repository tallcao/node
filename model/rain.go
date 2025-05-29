package model

import (
	"edge/utils"
	"time"

	"google.golang.org/protobuf/proto"
)

type RainSensor struct {
	raining *Payload_Metric

	guid string

	observer Observer

	Converter

	PassiveReporting *PassiveReporting
	IHeart
}

func NewRainSensor(guid string, c Converter, o Observer) *RainSensor {

	item := &RainSensor{
		raining: &Payload_Metric{
			Name:      proto.String("raining"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		Converter: c,

		PassiveReporting: &PassiveReporting{
			Interval: 60 * 5,
		},

		IHeart: new(Heart),

		observer: o,
	}

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
	i.SendFrame(data)

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

func (i *RainSensor) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.raining)
	i.observer.Update(i.guid, p)

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

func (i *RainSensor) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (i *RainSensor) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.raining)

	return p

}
