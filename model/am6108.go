package model

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type Am6108 struct {
	co2         *Payload_Metric
	hcho        *Payload_Metric
	pm25        *Payload_Metric
	temperature *Payload_Metric
	humidity    *Payload_Metric

	guid string

	converter Converter
	observer  Observer

	heart *Heart
}

func NewAm6108(guid string, c Converter, o Observer) *Am6108 {

	item := &Am6108{

		co2: &Payload_Metric{
			Name:      proto.String("co2"),
			Datatype:  proto.Uint32(uint32(DataType_UInt16)),
			Timestamp: proto.Uint64(0),
		},
		hcho: &Payload_Metric{
			Name:      proto.String("hcho"),
			Datatype:  proto.Uint32(uint32(DataType_UInt16)),
			Timestamp: proto.Uint64(0),
		},
		pm25: &Payload_Metric{
			Name:      proto.String("pm25"),
			Datatype:  proto.Uint32(uint32(DataType_UInt16)),
			Timestamp: proto.Uint64(0),
		},
		temperature: &Payload_Metric{
			Name:      proto.String("temperature"),
			Datatype:  proto.Uint32(uint32(DataType_Float)),
			Timestamp: proto.Uint64(0),
		},
		humidity: &Payload_Metric{
			Name:      proto.String("humidity"),
			Datatype:  proto.Uint32(uint32(DataType_UInt16)),
			Timestamp: proto.Uint64(0),
		},

		guid:      guid,
		converter: c,

		heart:    new(Heart),
		observer: o,
	}

	return item
}

func (d *Am6108) Response(data []byte) {
	if len(data) != 25 {
		return
	}

	ts := uint64(time.Now().UnixMicro())

	d.co2.Value = &Payload_Metric_IntValue{256*uint32(data[9]) + uint32(data[10])}
	*d.co2.Timestamp = ts

	d.hcho.Value = &Payload_Metric_IntValue{256*uint32(data[14]) + uint32(data[15])}
	*d.hcho.Timestamp = ts

	d.pm25.Value = &Payload_Metric_IntValue{256*uint32(data[7]) + uint32(data[8])}
	*d.hcho.Timestamp = ts

	d.temperature.Value = &Payload_Metric_FloatValue{(256*float32(data[11]) + float32(data[12]) - 100) / 10.0}
	*d.temperature.Timestamp = ts

	d.humidity.Value = &Payload_Metric_IntValue{uint32(data[13])}
	*d.humidity.Timestamp = ts

	d.notifyAll()
}

func (d *Am6108) Request(string, interface{}) {

}
func (d *Am6108) GetId() string {
	return d.guid
}
func (i *Am6108) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_AM6108
}

func (d *Am6108) GetConverter() Converter {
	return d.converter
}

func (i *Am6108) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.co2, i.hcho, i.pm25, i.temperature, i.humidity)

	i.observer.Update(i.guid, p)

}

func (i *Am6108) GetDevice485Setting() (uint32, byte, byte, byte) {
	return 9600, 0, 8, 1
}

func (i *Am6108) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *Am6108) HeartCheck() {
	i.heart.HeartCheck()
}

func (i *Am6108) IsConnected() bool {
	return i.heart.Conected
}

func (i *Am6108) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *Am6108) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.co2, i.hcho, i.pm25, i.temperature, i.humidity)

	return p

}
