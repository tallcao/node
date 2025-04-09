package model

import (
	"edge/utils"
	"time"

	"google.golang.org/protobuf/proto"
)

// stb3-125/R 不带计量断路器

type Breaker_STB3_125_R struct {
	lock *Payload_Metric
	on   *Payload_Metric

	guid string

	addr         byte
	observerList []Observer

	converter Converter

	heart *Heart
}

func NewBreaker_STB3_125_R(guid string, c Converter, o Observer) *Breaker_STB3_125_R {

	item := &Breaker_STB3_125_R{
		lock: &Payload_Metric{
			Name:      proto.String("lock"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},
		on: &Payload_Metric{
			Name:      proto.String("on"),
			Datatype:  proto.Uint32(uint32(DataType_Boolean)),
			Timestamp: proto.Uint64(0),
		},

		converter: c,

		heart: new(Heart),
	}

	item.register(o)

	if adapter, ok := c.(AddrAdapter); ok {
		item.addr = adapter.GetAddr()
	} else {
		item.addr = 0x01
	}

	return item
}

func (i *Breaker_STB3_125_R) Request(command string, params interface{}) {

	var data []byte
	data = append(data, i.addr)

	switch command {
	case "heartBeat":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "getStatus":
		data = append(data, 0x04, 0x00, 0x00, 0x00, 0x01)
	case "on":
		data = append(data, 0x05, 0x00, 0x01, 0xFF, 0x00)
	case "off":
		data = append(data, 0x05, 0x00, 0x01, 0x00, 0x00)
	case "toggle":
		v := byte(0xff)
		if i.on.GetBooleanValue() {
			v = 0
		}
		data = append(data, 0x05, 0x00, 0x01, v, 0x00)

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

func (d *Breaker_STB3_125_R) Response(data []byte) {

	if len(data) < 2 {
		return
	}

	on := d.on.GetBooleanValue()
	lock := d.lock.GetBooleanValue()

	if len(data) == 8 && data[1] == 0x05 {

		if data[4] == 0xFF {
			on = true
		}

		if data[4] == 0x00 {
			lock = false
		}

	}

	if len(data) == 7 && data[1] == 0x04 {

		if data[3] == 0x00 {
			lock = false

		}
		if data[3] == 0x01 {
			lock = true

		}
		if data[4] == 0xF0 {
			on = true

		}
		if data[4] == 0x0F {
			on = false

		}

	}

	onChanged := (on != d.on.GetBooleanValue())
	lockChanged := (lock != d.lock.GetBooleanValue())

	ts := uint64(time.Now().UnixMicro())
	d.on.Value = &Payload_Metric_BooleanValue{on}
	*d.on.Timestamp = ts

	d.lock.Value = &Payload_Metric_BooleanValue{lock}
	*d.lock.Timestamp = ts

	p := NewPayload()

	if onChanged {
		p.Metrics = append(p.Metrics, d.on)
	}

	if lockChanged {
		p.Metrics = append(p.Metrics, d.lock)

	}

	if len(p.Metrics) > 0 {
		for _, observer := range d.observerList {
			observer.Update(p)
		}
	}

}

func (i *Breaker_STB3_125_R) GetId() string {
	return i.guid
}
func (i *Breaker_STB3_125_R) GetType() DEVICE_TYPE {
	return DEVICE_TYPE_BREAKER_STB3_125_R
}

func (i *Breaker_STB3_125_R) GetConverter() Converter {
	return i.converter
}

func (i *Breaker_STB3_125_R) register(o Observer) {
	i.observerList = append(i.observerList, o)
}

func (i *Breaker_STB3_125_R) deregister(o Observer) {
	i.observerList = removeFromslice(i.observerList, o)
}

func (i *Breaker_STB3_125_R) notifyAll() {

	p := NewPayload()
	p.Metrics = append(p.Metrics, i.on, i.lock)

	for _, observer := range i.observerList {
		observer.Update(p)
	}

}

func (i *Breaker_STB3_125_R) HeartBeat() {
	i.heart.HeartBeat()
}

func (i *Breaker_STB3_125_R) HeartCheck() {
	i.heart.HeartCheck()
	if i.heart.Conected && i.heart.Changed() {
		i.Request("getStatus", nil)
	}
}

func (i *Breaker_STB3_125_R) IsConnected() bool {
	return i.heart.Conected
}

func (i *Breaker_STB3_125_R) ConnectedChanged() bool {
	return i.heart.Changed()
}

func (i *Breaker_STB3_125_R) DBirth() *Payload {
	p := NewPayload()

	p.Metrics = append(p.Metrics, i.lock, i.on)

	return p

}
