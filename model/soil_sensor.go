package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type SoilSensor struct {
	temperature  *Payload_Metric
	humidity     *Payload_Metric
	conductivity *Payload_Metric
	ph           *Payload_Metric

	guid string

	observer Observer

	Converter

	PassiveReporting *PassiveReporting

	IHeart
}

func NewSoilSensor(guid string, c Converter, o Observer) *SoilSensor {

	item := &SoilSensor{

		temperature: &Payload_Metric{
			Name:      proto.String("temperature"),
			Datatype:  proto.Uint32(uint32(DataType_Float)),
			Timestamp: proto.Uint64(0),
		},
		humidity: &Payload_Metric{
			Name:      proto.String("humidity"),
			Datatype:  proto.Uint32(uint32(DataType_Float)),
			Timestamp: proto.Uint64(0),
		},
		conductivity: &Payload_Metric{
			Name:      proto.String("conductivity"),
			Datatype:  proto.Uint32(uint32(DataType_Float)),
			Timestamp: proto.Uint64(0),
		},
		ph: &Payload_Metric{
			Name:      proto.String("ph"),
			Datatype:  proto.Uint32(uint32(DataType_Float)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		Converter: c,

		PassiveReporting: &PassiveReporting{
			Interval: 60 * 10,
		},

		IHeart:   new(Heart),
		observer: o,
	}

	return item
}

func (i *SoilSensor) Request(command string, params interface{}) {

	switch command {
	case "getStatus":
		data := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x04, 0x44, 0x09}
		i.SendFrame(data)
	case "setInterval":
		i.PassiveReporting.SetInterval(params)
		// i.notifyAll()
	default:
		return
	}
}

func (i *SoilSensor) Response(data []byte) {
	if len(data) != 13 {
		return
	}

	if data[1] != 0x03 {
		return
	}

	var humidity float32
	var temperature float32
	var conductivity float32
	var ph float32

	humidity = (float32(data[3])*256 + float32(data[4])) / 10

	flag := data[5] & 0x80

	if flag == 0 {
		temperature = (float32(data[5])*256 + float32(data[6])) / 10
	} else {
		temperature = (float32(data[5]^0xff)*256 + float32(data[6]^0xff) + 1) / 10
	}

	conductivity = float32(data[7])*256 + float32(data[8])
	ph = (float32(data[9])*256 + float32(data[10])) / 10

	ts := uint64(time.Now().UnixMicro())

	i.humidity.Value = &Payload_Metric_FloatValue{humidity}
	*i.humidity.Timestamp = ts

	i.temperature.Value = &Payload_Metric_FloatValue{temperature}
	*i.temperature.Timestamp = ts

	i.conductivity.Value = &Payload_Metric_FloatValue{conductivity}
	*i.conductivity.Timestamp = ts

	i.ph.Value = &Payload_Metric_FloatValue{ph}
	*i.ph.Timestamp = ts

	i.notifyAll()
}

func (i *SoilSensor) GetId() string {
	return i.guid
}

func (i *SoilSensor) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_SOIL_SENSOR
}

func (i *SoilSensor) notifyAll() {

	p := NewPayload()

	p.Metrics = append(p.Metrics, i.humidity, i.temperature, i.conductivity, i.ph)
	i.observer.Update(i.guid, p)

}

func (i *SoilSensor) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 4800, 0, 8, 1
}

func (i *SoilSensor) StartLoopRequest() {
	i.PassiveReporting.StartLoopRequest(i.Request, "getStatus")
}

func (i *SoilSensor) StopLoopRequest() {
	i.PassiveReporting.StopLoopRequest()
}

func (i *SoilSensor) HeartCheck() {
	i.IHeart.HeartCheck()
	if i.IHeart.IsConnected() && i.IHeart.ConnectedChanged() {
		i.Request("getStatus", nil)
	}
}

func (i *SoilSensor) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.humidity, i.temperature, i.conductivity, i.ph)

	return p

}
